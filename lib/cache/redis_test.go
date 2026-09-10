package cache

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"reflect"
	"testing"
)

func testRedis(t *testing.T) *RedisCache {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}
	rc := RedisCacheInit(&redis.Options{Addr: addr})
	if err := rc.rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable at %s: %v", addr, err)
	}
	return rc
}

func TestRedisSet(t *testing.T) {
	//rc := New("redis")
	rc := testRedis(t)
	err := rc.Set("123", "ddd", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("写入失败")
	}
}

func TestRedisGet(t *testing.T) {
	rc := testRedis(t)
	err := rc.Set("123", "451156", 300)
	if err != nil {
		t.Fatalf("写入失败")
	}
	res := ""
	err = rc.Get("123", &res)
	if err != nil {
		t.Fatalf("读取失败")
	}
	fmt.Println("res", res)
}

func TestRedisGetJson(t *testing.T) {
	rc := testRedis(t)
	type r struct {
		Aa string `json:"a"`
		B  string `json:"c"`
	}
	old := &r{
		Aa: "ab", B: "cdc",
	}
	err := rc.Set("1233", old, 300)
	if err != nil {
		t.Fatalf("写入失败")
	}

	res := &r{}
	err2 := rc.Get("1233", res)
	if err2 != nil {
		t.Fatalf("读取失败")
	}
	if !reflect.DeepEqual(res, old) {
		t.Fatalf("读取错误")
	}
	fmt.Println(res, res.Aa)
}

func BenchmarkRSet(b *testing.B) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		b.Skip("TEST_REDIS_ADDR is not set")
	}
	rc := RedisCacheInit(&redis.Options{Addr: addr})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rc.Set("123", "{dsv}", 1000)
	}
}

func BenchmarkRGet(b *testing.B) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		b.Skip("TEST_REDIS_ADDR is not set")
	}
	rc := RedisCacheInit(&redis.Options{Addr: addr})
	b.ResetTimer()
	v := ""
	for i := 0; i < b.N; i++ {
		rc.Get("123", &v)
	}
}
