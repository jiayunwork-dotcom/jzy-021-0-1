package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// chainNet 造一张关键通路随 i 变化的网络：主链 i+2 节，旁支 1 节。
func chainNet(i int) *NetworkInput {
	prefix := fmt.Sprintf("N%d_", i)
	acts := []ActivityInput{{ID: prefix + "S", Duration: d(1)}}
	prev := prefix + "S"
	for k := 1; k <= i+2; k++ {
		id := fmt.Sprintf("%sX%d", prefix, k)
		acts = append(acts, ActivityInput{ID: id, Duration: d(1), Predecessors: []string{prev}})
		prev = id
	}
	acts = append(acts, ActivityInput{ID: prefix + "P", Duration: d(1), Predecessors: []string{prefix + "S"}})
	return &NetworkInput{Activities: acts}
}

func wantChainPath(i int) []string {
	prefix := fmt.Sprintf("N%d_", i)
	path := []string{prefix + "S"}
	for k := 1; k <= i+2; k++ {
		path = append(path, fmt.Sprintf("%sX%d", prefix, k))
	}
	return path
}

// 多张网络并行计算：每份作业各自持有自己的关键通路，互不渗透。
func TestConcurrentJobsIsolation(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newServer(store))
	defer srv.Close()

	const N = 24
	type result struct {
		job *Job
		err error
	}
	results := make([]result, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body, _ := json.Marshal(chainNet(i))
			resp, err := http.Post(srv.URL+"/api/v1/jobs", "application/json", bytes.NewReader(body))
			if err != nil {
				results[i].err = err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				results[i].err = fmt.Errorf("状态码 %d", resp.StatusCode)
				return
			}
			var job Job
			if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
				results[i].err = err
				return
			}
			results[i].job = &job
		}(i)
	}
	wg.Wait()

	seenIDs := map[string]bool{}
	for i, r := range results {
		if r.err != nil {
			t.Fatalf("第 %d 张网络计算失败: %v", i, r.err)
		}
		job := r.job
		want := wantChainPath(i)
		if len(job.CriticalPaths) != 1 || !equalStrings(job.CriticalPaths[0], want) {
			t.Fatalf("第 %d 份作业关键通路被污染：得到 %v，期望 %v", i, job.CriticalPaths, want)
		}
		if seenIDs[job.ID] {
			t.Fatalf("作业编号 %s 重复发放", job.ID)
		}
		seenIDs[job.ID] = true

		// 取回后再核对一次：落库的也必须还是自己的关键通路。
		resp, err := http.Get(srv.URL + "/api/v1/jobs/" + job.ID)
		if err != nil {
			t.Fatal(err)
		}
		var got Job
		json.NewDecoder(resp.Body).Decode(&got)
		resp.Body.Close()
		if len(got.CriticalPaths) != 1 || !equalStrings(got.CriticalPaths[0], want) {
			t.Fatalf("取回的作业 %s 关键通路被污染：%v", job.ID, got.CriticalPaths)
		}
	}
}

func TestHTTPAPI(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newServer(store))
	defer srv.Close()

	// 内置示范网络可取。
	resp, err := http.Get(srv.URL + "/api/v1/demo")
	if err != nil {
		t.Fatal(err)
	}
	var demo NetworkInput
	json.NewDecoder(resp.Body).Decode(&demo)
	resp.Body.Close()
	if len(demo.Activities) != 8 {
		t.Fatalf("示范网络应有 8 道工序，得到 %d", len(demo.Activities))
	}

	// 环路返回带类型的错误。
	bad := `{"activities":[{"id":"A","duration":1,"predecessors":["B"]},{"id":"B","duration":1,"predecessors":["A"]}]}`
	resp, err = http.Post(srv.URL+"/api/v1/jobs", "application/json", strings.NewReader(bad))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("环路应返回 400，得到 %d", resp.StatusCode)
	}
	var errBody struct {
		Error APIError `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&errBody)
	resp.Body.Close()
	if errBody.Error.Type != ErrTypeCycle {
		t.Fatalf("期望 cycle 错误，得到 %q", errBody.Error.Type)
	}

	// 非法 JSON。
	resp, err = http.Post(srv.URL+"/api/v1/jobs", "application/json", strings.NewReader("{oops"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法 JSON 应返回 400，得到 %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 正常提交 → 201，再按编号取回。
	body, _ := json.Marshal(demoNetwork())
	resp, err = http.Post(srv.URL+"/api/v1/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d", resp.StatusCode)
	}
	var job Job
	json.NewDecoder(resp.Body).Decode(&job)
	resp.Body.Close()
	if job.ID == "" {
		t.Fatal("响应应带作业编号")
	}
	if job.ProjectDuration != 24 {
		t.Fatalf("示范网络工期应为 24，得到 %v", job.ProjectDuration)
	}

	resp, err = http.Get(srv.URL + "/api/v1/jobs/" + job.ID)
	if err != nil {
		t.Fatal(err)
	}
	var got Job
	json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got.ID != job.ID || got.ProjectDuration != 24 {
		t.Fatal("取回的作业与提交时不一致")
	}

	// 不存在与非法编号 → 404。
	for _, id := range []string{"job-999999", "not-a-job", "..%2F..%2Fetc"} {
		resp, err = http.Get(srv.URL + "/api/v1/jobs/" + id)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("编号 %q 应返回 404，得到 %d", id, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
