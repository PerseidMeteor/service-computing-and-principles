package yrpc_test

import (
	"os"
	"testing"
)

// 定义被测试的函数 Add()
func Add(a, b int) int {
	return a + b
}

// 定义测试用例 TestAdd
func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("1+2 should be equal to 3")
	}
	if Add(10, 20) != 30 {
		t.Error("10+20 should be equal to 30")
	}
}

// 运行所有测试用例
func TestMain(m *testing.M) {
	println("set up")
	code := m.Run()
	println("tear down")
	os.Exit(code)
}
