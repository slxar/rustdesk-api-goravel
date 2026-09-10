package jwt

import (
	"testing"
	"time"
)

const testJWTKey = "test-only-jwt-signing-key"

// 测试token生成
func TestGenerateToken(t *testing.T) {
	jwtService := NewJwt(testJWTKey, time.Second*1000)
	token := jwtService.GenerateToken(1)
	if token == "" {
		t.Fatal("token生成失败")
	}
}

// 测试token解析
func TestParseToken(t *testing.T) {
	jwtService := NewJwt(testJWTKey, time.Second*1000)
	token := jwtService.GenerateToken(999)
	if token == "" {
		t.Fatal("token生成失败")
	}
	uid, err := jwtService.ParseToken(token)
	if err != nil {

		t.Fatal("token解析失败", err)
	}
	if uid != 999 {
		t.Fatal("token解析失败")
	}
}

func BenchmarkJwtService_GenerateToken(b *testing.B) {
	jwtService := NewJwt(testJWTKey, time.Second*1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jwtService.GenerateToken(999)
	}
}

func BenchmarkJwtService_ParseToken(b *testing.B) {
	jwtService := NewJwt(testJWTKey, time.Second*1000)
	token := jwtService.GenerateToken(999)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jwtService.ParseToken(token)
	}

}
