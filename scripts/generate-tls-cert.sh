#!/bin/bash
# Waterflow TLS Certificate Generation Script
#
# 生成 TLS 证书用于 Waterflow HTTPS 配置
# 支持开发环境 (自签名) 和生产环境 (CSR)
#
# 使用方法:
#   开发环境: ./generate-tls-cert.sh development localhost
#   测试环境: ./generate-tls-cert.sh development waterflow.test.com
#   生产环境: ./generate-tls-cert.sh production waterflow.example.com
#
# 参数:
#   $1 - 环境类型: development (自签名) | production (生成 CSR)
#   $2 - 域名
#   $3 - 输出目录 (可选,默认: ./tls)
#   $4 - 有效期天数 (可选,默认: 365)

set -e

# ========================================
# 参数解析
# ========================================
ENV_TYPE=${1:-development}
DOMAIN=${2:-localhost}
OUTPUT_DIR=${3:-./tls}
DAYS=${4:-365}

# ========================================
# 颜色输出
# ========================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# ========================================
# 验证参数
# ========================================
if [[ ! "$ENV_TYPE" =~ ^(development|production)$ ]]; then
    error "无效的环境类型: $ENV_TYPE (必须是 development 或 production)"
fi

if [ -z "$DOMAIN" ]; then
    error "域名不能为空"
fi

# ========================================
# 创建输出目录
# ========================================
mkdir -p "$OUTPUT_DIR"

echo "========================================="
echo " Waterflow TLS 证书生成"
echo "========================================="
echo "环境类型: $ENV_TYPE"
echo "域名:     $DOMAIN"
echo "输出目录: $OUTPUT_DIR"
echo "有效期:   $DAYS 天"
echo "========================================="
echo ""

# ========================================
# 生成开发环境自签名证书
# ========================================
if [ "$ENV_TYPE" == "development" ]; then
    info "生成自签名证书 (开发/测试环境)..."
    
    # 生成私钥
    openssl genrsa -out "$OUTPUT_DIR/key.pem" 2048
    info "✓ 私钥已生成: $OUTPUT_DIR/key.pem"
    
    # 生成证书签名请求
    openssl req -new \
        -key "$OUTPUT_DIR/key.pem" \
        -out "$OUTPUT_DIR/cert.csr" \
        -subj "/CN=$DOMAIN/O=Waterflow/C=US" \
        -addext "subjectAltName=DNS:$DOMAIN,DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"
    info "✓ CSR 已生成: $OUTPUT_DIR/cert.csr"
    
    # 生成自签名证书
    openssl x509 -req \
        -in "$OUTPUT_DIR/cert.csr" \
        -signkey "$OUTPUT_DIR/key.pem" \
        -out "$OUTPUT_DIR/cert.pem" \
        -days "$DAYS" \
        -copy_extensions copyall
    info "✓ 证书已生成: $OUTPUT_DIR/cert.pem"
    
    # 设置文件权限
    chmod 600 "$OUTPUT_DIR/key.pem"
    chmod 644 "$OUTPUT_DIR/cert.pem"
    chmod 644 "$OUTPUT_DIR/cert.csr"
    
    # 删除 CSR (可选)
    # rm "$OUTPUT_DIR/cert.csr"
    
    echo ""
    info "自签名证书生成完成!"
    echo ""
    echo "文件列表:"
    ls -lh "$OUTPUT_DIR"/*.pem
    echo ""
    
    # 显示证书信息
    info "证书信息:"
    openssl x509 -in "$OUTPUT_DIR/cert.pem" -noout -subject -issuer -dates
    echo ""
    
    # 配置示例
    echo "========================================="
    echo " 配置示例"
    echo "========================================="
    echo ""
    echo "config.yaml:"
    echo "  https:"
    echo "    enabled: true"
    echo "    cert_file: $OUTPUT_DIR/cert.pem"
    echo "    key_file: $OUTPUT_DIR/key.pem"
    echo "    min_version: \"1.2\""
    echo ""
    echo "验证 HTTPS:"
    echo "  curl -k https://localhost:8443/health"
    echo ""
    warn "⚠️  自签名证书仅用于开发/测试环境,生产环境请使用 CA 签名证书"
    echo ""

# ========================================
# 生成生产环境 CSR
# ========================================
elif [ "$ENV_TYPE" == "production" ]; then
    info "生成证书签名请求 (CSR) 用于生产环境..."
    
    # 生成私钥
    openssl genrsa -out "$OUTPUT_DIR/key.pem" 4096
    info "✓ 私钥已生成: $OUTPUT_DIR/key.pem (4096 位)"
    
    # 生成 CSR
    openssl req -new \
        -key "$OUTPUT_DIR/key.pem" \
        -out "$OUTPUT_DIR/cert.csr" \
        -subj "/CN=$DOMAIN/O=Waterflow/C=US"
    info "✓ CSR 已生成: $OUTPUT_DIR/cert.csr"
    
    # 设置文件权限
    chmod 600 "$OUTPUT_DIR/key.pem"
    chmod 644 "$OUTPUT_DIR/cert.csr"
    
    echo ""
    info "CSR 生成完成!"
    echo ""
    echo "文件列表:"
    ls -lh "$OUTPUT_DIR"/*.{pem,csr}
    echo ""
    
    # 显示 CSR 信息
    info "CSR 信息:"
    openssl req -in "$OUTPUT_DIR/cert.csr" -noout -text | grep -A1 "Subject:"
    echo ""
    
    echo "========================================="
    echo " 下一步操作"
    echo "========================================="
    echo ""
    echo "1. 提交 CSR 到 CA:"
    echo "   - Let's Encrypt: certbot certonly --csr $OUTPUT_DIR/cert.csr"
    echo "   - 企业 CA: 通过企业流程提交 CSR"
    echo ""
    echo "2. 获取签名证书后,保存为 $OUTPUT_DIR/cert.pem"
    echo ""
    echo "3. 配置 Waterflow:"
    echo "   https:"
    echo "     enabled: true"
    echo "     cert_file: $OUTPUT_DIR/cert.pem"
    echo "     key_file: $OUTPUT_DIR/key.pem"
    echo ""
    echo "4. (可选) 配置 CA 证书:"
    echo "     ca_cert: $OUTPUT_DIR/ca.pem"
    echo ""
    echo "========================================="
    echo " Let's Encrypt 快速生成"
    echo "========================================="
    echo ""
    echo "如果使用 Let's Encrypt:"
    echo ""
    echo "# 安装 certbot"
    echo "sudo apt-get install certbot"
    echo ""
    echo "# 生成证书"
    echo "sudo certbot certonly --standalone -d $DOMAIN"
    echo ""
    echo "# 证书位置"
    echo "# /etc/letsencrypt/live/$DOMAIN/fullchain.pem"
    echo "# /etc/letsencrypt/live/$DOMAIN/privkey.pem"
    echo ""
    echo "# 复制到输出目录"
    echo "sudo cp /etc/letsencrypt/live/$DOMAIN/fullchain.pem $OUTPUT_DIR/cert.pem"
    echo "sudo cp /etc/letsencrypt/live/$DOMAIN/privkey.pem $OUTPUT_DIR/key.pem"
    echo "sudo chown \$USER:USER $OUTPUT_DIR/*.pem"
    echo ""
fi

info "✅ 完成!"
