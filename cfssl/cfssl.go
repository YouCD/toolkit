package cfssl

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"

	"github.com/cloudflare/cfssl/cli"
	"github.com/cloudflare/cfssl/cli/genkey"
	"github.com/cloudflare/cfssl/cli/sign"
	cfsslConfigStruct "github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	cfsslLog "github.com/cloudflare/cfssl/log"
	"github.com/cloudflare/cfssl/signer"
	"github.com/youcd/toolkit/file"
)

const (
	CAConfigFileData = `{"signing":{"default":{"expiry":"876000h"},"profiles":{"www":{"expiry":"876000h","usages":["signing","key encipherment","server auth"]}}}}`
)

var (
	CACertFile    string
	CACSRFile     string
	CAKeyFile     string
	CAConfigFile  string
	Server        string
	ServerKey     string
	ServerCSR     string
	ServerCSRJson string
)

// signerFromConfig
//
//	@Description:加载Signer配置
//	@param caFile
//	@param caKeyFile
//	@param configFile
//	@return cli.ConfigEdit
//	@return signer.Signer
//	@return error
//
//nolint:ireturn
func signerFromConfig(caFile, caKeyFile string) (*cli.Config, signer.Signer, error) {
	conf, err := cfsslConfigStruct.LoadConfig([]byte(CAConfigFileData))
	if err != nil {
		return nil, nil, fmt.Errorf("加载配置文件,err:%w", err)
	}

	c := cli.Config{
		CAFile:    caFile,
		CAKeyFile: caKeyFile,
		CFG:       conf,
		Profile:   "www",
	}

	s, err := sign.SignerFromConfig(c)
	if err != nil {
		return nil, nil, fmt.Errorf("获取签名器,err:%w", err)
	}
	return &c, s, nil
}

func initCAFiles(saveDir string) {
	// ca.pem
	CACertFile = filepath.Join(saveDir, "ca.pem")
	// ca.csr
	CACSRFile = filepath.Join(saveDir, "ca.csr")
	// ca-key.pem
	CAKeyFile = filepath.Join(saveDir, "ca-key.pem")
	// ca-config.json
	CAConfigFile = filepath.Join(saveDir, "ca-config.json")

	Server = filepath.Join(saveDir, "server.crt")
	ServerKey = filepath.Join(saveDir, "server.key")
	ServerCSR = filepath.Join(saveDir, "server.csr")
	ServerCSRJson = filepath.Join(saveDir, "csr.json")
}

// Signe  证书签发
func Signe(saveDir string, externalIP string, domainName string) (map[string][]byte, error) {
	//  cfss 设置日志级别
	cfsslLog.Level = cfsslLog.LevelFatal

	// 初始化CA
	_, err := initCA(saveDir, domainName)
	if err != nil {
		return nil, fmt.Errorf("初始化CA,err:%w", err)
	}

	//  证书配置文件
	caConfig, s, err := signerFromConfig(CACertFile, CAKeyFile)
	if err != nil {
		return nil, fmt.Errorf("证书配置文件,err:%w", err)
	}

	// 证书请求
	req, err := newCSRReq(externalIP, domainName)
	if err != nil {
		return nil, fmt.Errorf("证书请求,err:%w", err)
	}

	g := &csr.Generator{Validator: genkey.Validator}

	csrBytes, key, err := g.ProcessRequest(&req)
	if err != nil {
		return nil, fmt.Errorf("签发请求,err:%w", err)
	}

	signReq := signer.SignRequest{
		Request: string(csrBytes),
		Hosts:   signer.SplitHosts(caConfig.Hostname),
		Profile: caConfig.Profile,
	}

	cert, err := s.Sign(signReq)
	if err != nil {
		return nil, fmt.Errorf("证书签发,err:%w", err)
	}

	writeFiles := make(map[string][]byte)

	//nolint:errchkjson
	var caConfigData interface{}
	_ = json.Unmarshal([]byte(CAConfigFileData), &caConfigData)
	data, _ := json.MarshalIndent(caConfigData, "", "    ")
	writeFiles[CAConfigFile] = data

	writeFiles[Server] = cert
	writeFiles[ServerKey] = key
	writeFiles[ServerCSR] = csrBytes
	//nolint:errchkjson
	bytes, _ := json.MarshalIndent(req, "", "    ")

	writeFiles[ServerCSRJson] = bytes
	for fileName, data := range writeFiles {
		err = file.Write(data, fileName, 0o644)
		if err != nil {
			return nil, fmt.Errorf("写入文件:%s,err:%w", fileName, err)
		}
	}

	return writeFiles, nil
}

// newCSRReq
//
//	@Description: 证书请求
//	@return csr.CertificateRequest
//	@return error
func newCSRReq(externalIP string, externalDomainName string) (csr.CertificateRequest, error) {
	// 1. 创建基本的 CSR 请求
	req := csr.CertificateRequest{
		KeyRequest: csr.NewKeyRequest(), // 默认 RSA 2048
		CN:         externalDomainName,
		Names: []csr.Name{
			{
				C:  "CN",
				ST: "HuNan",
				L:  "ChangSha",
			},
		},
	}

	// 2. 构建 hosts 列表
	hosts := []string{
		externalDomainName,
		"*." + externalDomainName,
		"127.0.0.1",
		externalIP,
	}

	// 3. 解析并设置 hosts（自动识别 IP 和域名）
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			req.Hosts = append(req.Hosts, ip.String())
		} else {
			req.Hosts = append(req.Hosts, h)
		}
	}

	// 4. 可选：自定义 KeyRequest 参数
	// 如果需要自定义算法或大小
	// req.KeyRequest = &csr.BasicKeyRequest{
	//     Algo: "rsa",
	//     Size: 2048,
	// }

	// 5. 验证 CSR 请求的合法性
	if err := validateCSRRequest(&req); err != nil {
		return req, fmt.Errorf("invalid CSR request: %w", err)
	}

	return req, nil
}

// 验证 CSR 请求
func validateCSRRequest(req *csr.CertificateRequest) error {
	if req.CN == "" {
		return fmt.Errorf("CN cannot be empty")
	}
	if len(req.Hosts) == 0 {
		return fmt.Errorf("at least one host or IP required")
	}
	return nil
}

// initCA
//
//	@Description: 初始化CA
//	@return map[string][]byte
//	@return error
func initCA(saveDir string, hostname string) (map[string][]byte, error) {
	initCAFiles(saveDir)

	// 设置过期时间
	invalidCAConfig := csr.CAConfig{
		PathLength: 2,
		Expiry:     "876000h",
	}

	req := &csr.CertificateRequest{
		Names: []csr.Name{{
			C:  "CN",
			ST: "HuNan",
			L:  "ChangSha",
			O:  "FireCloud",
			OU: "DEVOPS",
		}},
		CN:         hostname,
		Hosts:      []string{hostname, hostname},
		KeyRequest: &csr.KeyRequest{A: "rsa", S: 2048},
		CA:         &invalidCAConfig,
	}

	// 初始化CA
	CACert, CACsr, CAKey, err := initca.New(req)
	if err != nil {
		return nil, fmt.Errorf("初始化CA,err:%w", err)
	}

	writeFiles := make(map[string][]byte)

	writeFiles[CACertFile] = CACert
	// ca.csr
	writeFiles[CACSRFile] = CACsr
	// ca-key.pem
	writeFiles[CAKeyFile] = CAKey

	// ca-config.json
	writeFiles[CAConfigFile] = []byte(CAConfigFileData)

	for fileName, data := range writeFiles {
		err = file.Write(data, fileName, 0o644)
		if err != nil {
			return nil, fmt.Errorf("写入文件:%s,err:%w", fileName, err)
		}
	}
	return writeFiles, nil
}
