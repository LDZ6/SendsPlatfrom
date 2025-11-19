#!/bin/bash

# 生成各服务的Dockerfile脚本

SERVICES=(
    "gateway:8080"
    "boBing:8081"
    "user:10002"
    "school:10004"
    "yearBill:10005"
)

TEMPLATE_FILE="docker/Dockerfile.template"

for service_info in "${SERVICES[@]}"; do
    IFS=':' read -r service_name port <<< "$service_info"
    
    echo "生成 $service_name 服务的Dockerfile..."
    
    # 创建服务目录的Dockerfile
    sed "s/{{SERVICE_NAME}}/$service_name/g; s/{{PORT}}/$port/g" "$TEMPLATE_FILE" > "app/$service_name/cmd/Dockerfile"
    
    echo "✅ $service_name 服务的Dockerfile已生成"
done

echo "🎉 所有服务的Dockerfile已生成完成！"
