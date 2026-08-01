package secretref

import (
	"encoding/json"
	"os"
	"testing"
)

// 正常路径：合法引用名与普通环境变量名通过。
func TestRefNamesValid(t *testing.T) {
	for _, name := range []string{"IDENTITY_SIGNING_KEY_REF", "DB_FIELD_ENCRYPTION_KEY_REF", "EMAIL_OTP_PEPPER_REF"} {
		if !IsRefName(name) {
			t.Fatalf("%q 应识别为密钥引用", name)
		}
		if err := ValidateRefName(name); err != nil {
			t.Fatalf("%q 应通过: %v", name, err)
		}
	}
	if err := ValidateEnvVarName("DATABASE_URL"); err != nil {
		t.Fatalf("DATABASE_URL 应通过: %v", err)
	}
}

// 异常路径：非 _REF 结尾、非法字符必须拒绝。
func TestRefNamesRejected(t *testing.T) {
	for _, name := range []string{"IDENTITY_SIGNING_KEY", "lower_ref", "1BAD_REF", "BAD-REF", ""} {
		if err := ValidateRefName(name); err == nil {
			t.Fatalf("引用名 %q 必须拒绝", name)
		}
	}
}

// 正常路径：密钥引用表合法（值均为环境变量名，无内联密钥）。
func TestValidateRefsValid(t *testing.T) {
	// #nosec G101 -- 以下均为环境变量引用名（非真实密钥），用于校验 REF 引用契约（SEC-012）
	refs := map[string]string{
		"identity_signing_key":   "IDENTITY_SIGNING_KEY_REF",
		"field_encryption_key":   "DB_FIELD_ENCRYPTION_KEY_REF",
		"database_url":           "DATABASE_URL",
		"object_storage_keys":    "OBJECT_STORAGE_ACCESS_KEY",
		"payment_webhook_secret": "PAYMENT_WEBHOOK_SIGNING_SECRET",
		"otp_pepper":             "EMAIL_OTP_PEPPER_REF",
	}
	if err := ValidateRefs(refs); err != nil {
		t.Fatalf("合法引用表应通过: %v", err)
	}
}

// 异常路径：空表、空值、非法变量名、内联真实密钥必须拒绝（fail-closed）。
func TestValidateRefsRejected(t *testing.T) {
	raw, err := os.ReadFile("../../fixtures/synthetic/secret-ref-samples/inline-secret-samples.json")
	if err != nil {
		t.Fatalf("读取合成内联密钥样本失败: %v", err)
	}
	var doc struct {
		Synthetic bool `json:"synthetic"`
		Samples   []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"samples"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析合成内联密钥样本失败: %v", err)
	}
	if !doc.Synthetic || len(doc.Samples) == 0 {
		t.Fatal("合成内联密钥样本必须标记 synthetic: true 且非空")
	}
	cases := map[string]map[string]string{
		"空表":    {},
		"空值":    {"identity_signing_key": ""},
		"非法变量名": {"identity_signing_key": "identity signing key"},
	}
	for _, sample := range doc.Samples {
		cases["内联密钥("+sample.Name+")"] = map[string]string{"identity_signing_key": sample.Value}
	}
	for name, refs := range cases {
		if err := ValidateRefs(refs); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 正常路径：展示掩码不泄露完整密钥。
func TestMaskSecret(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"abcd":             "****",
		"secret1234567890": "****7890",
	}
	for in, want := range cases {
		if got := MaskSecret(in); got != want {
			t.Fatalf("MaskSecret(%q) = %q，期望 %q", in, got, want)
		}
	}
	if got := MaskSecret("real-secret-value-1234"); got == "real-secret-value-1234" {
		t.Fatal("掩码不得等于原文")
	}
}

// 幂等性：引用校验与掩码结果确定可重复（DoD 第 3 条）。
func TestSecretsIdempotent(t *testing.T) {
	refs := map[string]string{"identity_signing_key": "IDENTITY_SIGNING_KEY_REF"}
	firstMask := MaskSecret("abcdef1234")
	for i := 0; i < 3; i++ {
		if err := ValidateRefs(refs); err != nil {
			t.Fatalf("引用校验必须幂等通过: %v", err)
		}
		if MaskSecret("abcdef1234") != firstMask {
			t.Fatal("掩码结果必须确定")
		}
	}
}
