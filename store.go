package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Store 本地文件作业库：每份作业一个 JSON 文件，不另起数据库进程。
type Store struct {
	dir  string
	mu   sync.Mutex
	next int
}

var jobIDPattern = regexp.MustCompile(`^job-[0-9]{6}$`)

// NewStore 打开（必要时创建）文件库，并扫描已有作业恢复编号计数，
// 重启后编号不回退。
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, next: 1}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "job-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(name, "job-%06d.json", &n); err == nil && n >= s.next {
			s.next = n + 1
		}
	}
	return s, nil
}

// Save 为作业分配编号并原子写入文件库。
func (s *Store) Save(job *Job) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("job-%06d", s.next)
	job.ID = id
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := filepath.Join(s.dir, id+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, id+".json")); err != nil {
		return "", err
	}
	s.next++
	return id, nil
}

// Get 按编号取回作业；编号格式非法或不存在都返回 not_found。
func (s *Store) Get(id string) (*Job, error) {
	if !jobIDPattern.MatchString(id) {
		return nil, newError(ErrTypeNotFound, "作业 %q 不存在", id)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newError(ErrTypeNotFound, "作业 %q 不存在", id)
		}
		return nil, err
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}
