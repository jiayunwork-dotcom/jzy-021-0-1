package main

import (
	"errors"
	"testing"
)

// 文件库：保存/取回、重开后编号不回退、非法编号报 not_found。
func TestStorePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s1, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	id1, err := s1.Save(&Job{ProjectDuration: 1})
	if err != nil {
		t.Fatal(err)
	}

	s2, err := NewStore(dir) // 重新打开同一个库
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s2.Save(&Job{ProjectDuration: 2})
	if err != nil {
		t.Fatal(err)
	}
	if id1 == id2 {
		t.Fatalf("重新打开后编号回退：%s 与 %s", id1, id2)
	}

	got, err := s2.Get(id1)
	if err != nil {
		t.Fatalf("取回旧作业失败: %v", err)
	}
	if got.ProjectDuration != 1 {
		t.Fatalf("旧作业内容不对：%v", got.ProjectDuration)
	}

	var apiErr *APIError
	if _, err := s2.Get("job-999999"); !errors.As(err, &apiErr) || apiErr.Type != ErrTypeNotFound {
		t.Fatalf("不存在的编号应报 not_found，得到 %v", err)
	}
	if _, err := s2.Get("../etc/passwd"); !errors.As(err, &apiErr) || apiErr.Type != ErrTypeNotFound {
		t.Fatalf("非法编号应报 not_found，得到 %v", err)
	}
}
