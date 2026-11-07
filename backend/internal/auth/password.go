package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = bcrypt.DefaultCost

// HashPassword 生成用于 ADMIN_PASSWORD_HASH 的 bcrypt 哈希。
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验明文密码与 bcrypt 哈希是否匹配。
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// ValidateHash 检查字符串是否为合法的 bcrypt 哈希。
func ValidateHash(hash string) error {
	_, err := bcrypt.Cost([]byte(hash))
	return err
}
