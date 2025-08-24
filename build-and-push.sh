#!/bin/bash
###
 # @Date: 2025-08-24 21:12:24
 # @LastEditTime: 2025-08-24 21:15:11
### 

# 设置变量
IMAGE_NAME="pikachun"
REGISTRY="ghcr.io"
ORGANIZATION="mcp-zero"
FULL_IMAGE_NAME="${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}"

# 检查是否提供了 GitHub Token
if [ -z "$GITHUB_TOKEN" ]; then
  echo "请设置 GITHUB_TOKEN 环境变量"
  echo "例如: export GITHUB_TOKEN=your_github_token"
  exit 1
fi

# 构建 Docker 镜像
echo "正在构建 Docker 镜像..."
docker build -t ${IMAGE_NAME} .

# 给镜像打标签
echo "正在给镜像打标签..."
docker tag ${IMAGE_NAME} ${FULL_IMAGE_NAME}:latest

# 登录到 GitHub Container Registry
echo "正在登录到 GitHub Container Registry..."
echo $GITHUB_TOKEN | docker login ${REGISTRY} -u ${ORGANIZATION} --password-stdin

# 推送镜像到 GitHub Packages
echo "正在推送镜像到 GitHub Packages..."
docker push ${FULL_IMAGE_NAME}:latest

echo "镜像已成功推送到 GitHub Packages: ${FULL_IMAGE_NAME}:latest"