#!/bin/bash

# 项目清理脚本 - 删除冗余文件和重复代码

echo "🧹 开始清理项目冗余..."

# 1. 删除重复的Dockerfile
echo "删除重复的Dockerfile..."
find . -name "Dockerfile.optimized" -delete
echo "✅ 已删除重复的Dockerfile.optimized文件"

# 2. 删除重复的配置文件
echo "检查重复的配置文件..."
if [ -f "config/test_config.yml" ]; then
    echo "⚠️  发现测试配置文件，建议保留用于开发环境"
fi

# 3. 清理临时文件
echo "清理临时文件..."
find . -name "*.tmp" -delete
find . -name "*.log" -delete
find . -name ".DS_Store" -delete
echo "✅ 已清理临时文件"

# 4. 检查重复的代码文件
echo "检查重复的代码文件..."
duplicate_files=$(find . -name "*.go" -exec basename {} \; | sort | uniq -d)
if [ -n "$duplicate_files" ]; then
    echo "⚠️  发现重复的Go文件:"
    echo "$duplicate_files"
else
    echo "✅ 未发现重复的Go文件"
fi

# 5. 检查大文件
echo "检查大文件..."
large_files=$(find . -type f -size +1M -not -path "./.git/*" -not -path "./node_modules/*")
if [ -n "$large_files" ]; then
    echo "⚠️  发现大文件:"
    echo "$large_files"
else
    echo "✅ 未发现大文件"
fi

# 6. 统计项目文件
echo "📊 项目文件统计:"
echo "Go文件数量: $(find . -name "*.go" | wc -l)"
echo "Markdown文件数量: $(find . -name "*.md" | wc -l)"
echo "Dockerfile数量: $(find . -name "Dockerfile" | wc -l)"
echo "配置文件数量: $(find . -name "*.yml" -o -name "*.yaml" | wc -l)"

echo "🎉 项目清理完成！"
