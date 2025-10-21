#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
知识库初始化脚本 - 仅使用 REST API
将商品信息、常见问题和文档链接插入到 Chroma 向量数据库
"""

import os
import sys
import time
import requests
import json
import hashlib
import random

# 配置
DASHSCOPE_API_KEY = os.getenv("DASHSCOPE_API_KEY")
CHROMA_HOST = os.getenv("CHROMA_HOST", "localhost")
CHROMA_PORT = os.getenv("CHROMA_PORT", "8000")
COLLECTION_NAME = "shop_knowledge"

# DashScope Embedding API
EMBEDDING_API_URL = "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"
EMBEDDING_MODEL = "text-embedding-v2"


def generate_embedding(text: str) -> list:
    """生成文本的嵌入向量"""
    try:
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {DASHSCOPE_API_KEY}"
        }
        
        data = {
            "model": EMBEDDING_MODEL,
            "input": {
                "texts": [text]
            }
        }
        
        response = requests.post(EMBEDDING_API_URL, json=data, headers=headers, timeout=10)
        
        if response.status_code != 200:
            print(f"❌ Embedding API 错误 (状态码 {response.status_code})")
            raise Exception(response.text)
        
        result = response.json()
        
        if "output" in result and "embeddings" in result["output"]:
            embeddings = result["output"]["embeddings"]
            if len(embeddings) > 0:
                return embeddings[0]["embedding"]
        
        raise Exception(f"无效的响应结构")
        
    except Exception as e:
        print(f"⚠️  Embedding API 失败: {str(e)[:100]}")
        # 本地回退:生成确定性向量
        hash_obj = hashlib.md5(text.encode('utf-8'))
        hash_int = int(hash_obj.hexdigest(), 16)
        random.seed(hash_int)
        embedding = [random.uniform(-1, 1) for _ in range(1536)]
        norm = sum(x**2 for x in embedding) ** 0.5
        embedding = [x / norm for x in embedding]
        return embedding


def init_knowledge_base():
    """初始化知识库"""
    
    if not DASHSCOPE_API_KEY:
        print("❌ 错误: 未设置 DASHSCOPE_API_KEY 环境变量")
        sys.exit(1)
    
    print("🚀 开始初始化知识库...")
    print(f"   - Chroma 地址: http://{CHROMA_HOST}:{CHROMA_PORT}")
    
    chroma_url = f"http://{CHROMA_HOST}:{CHROMA_PORT}"
    tenant = "default_tenant"
    database = "default_database"
    
    # 等待 Chroma 启动
    max_retries = 30
    for i in range(max_retries):
        try:
            response = requests.get(f"{chroma_url}/api/v2/auth/identity", timeout=2)
            identity = response.json()
            tenant = identity.get("tenant", "default_tenant")
            databases = identity.get("databases", ["default_database"])
            database = databases[0] if databases else "default_database"
            print(f"✅ Chroma 已就绪 (租户: {tenant}, 数据库: {database})")
            break
        except:
            pass
        
        if i == max_retries - 1:
            print(f"❌ 无法连接到 Chroma")
            sys.exit(1)
        
        print(f"⏳ 等待 Chroma ({i+1}/{max_retries})")
        time.sleep(1)
    
    # 创建集合
    print(f"\n📝 创建集合...")
    collection_id = None
    try:
        response = requests.post(
            f"{chroma_url}/api/v2/tenants/{tenant}/databases/{database}/collections",
            json={"name": COLLECTION_NAME},
            timeout=5
        )
        if response.status_code in [200, 201]:
            result = response.json()
            collection_id = result.get("id")
            print(f"✅ 集合已创建: {COLLECTION_NAME} (ID: {collection_id})")
        else:
            print(f"   状态码: {response.status_code}")
    except Exception as e:
        print(f"⚠️  创建集合异常: {str(e)[:50]}")
    
    # 如果集合不是新创建的,查询现有集合
    if not collection_id:
        print(f"📝 查询现有集合...")
        try:
            response = requests.get(
                f"{chroma_url}/api/v2/tenants/{tenant}/databases/{database}/collections",
                timeout=5
            )
            if response.status_code == 200:
                collections = response.json()
                for col in collections:
                    if col.get("name") == COLLECTION_NAME:
                        collection_id = col.get("id")
                        print(f"✅ 找到现有集合: {COLLECTION_NAME} (ID: {collection_id})")
                        break
            if not collection_id:
                print(f"❌ 无法找到或创建集合")
                sys.exit(1)
        except Exception as e:
            print(f"❌ 查询集合失败: {e}")
            sys.exit(1)
    
    # 知识数据
    knowledge_data = [
        {"id": "product_bike_1", "text": "山地自行车 Pro X1 是一款专业级山地自行车,售价 3999 元。特点:配备27速变速系统,前后避震,碳纤维车架,轮径27.5英寸,重量仅12kg。适合越野和山地骑行,性能出色,操控灵活。", "category": "商品信息"},
        {"id": "product_bike_2", "text": "公路自行车 Speed R3 是一款高性能公路自行车,售价 2899 元。特点:轻量化设计,铝合金车架,700C轮径,18速变速系统,重量仅9kg。适合长途骑行和竞速,速度快,续航能力强。", "category": "商品信息"},
        {"id": "product_bike_3", "text": "城市通勤自行车 City Easy 是一款舒适的城市自行车,售价 1299 元。特点:钢制车架,26英寸轮径,单速设计,配备货架和挡泥板。适合日常代步和短途通勤,价格实惠,维护简单。", "category": "商品信息"},
        {"id": "product_helmet_1", "text": "专业骑行头盔 SafeRide Pro,售价 299 元。特点:轻量化设计(仅280g),通风良好,符合 CE/CPSC 国际安全标准,内置LED尾灯。尺寸可调节(54-62cm),提高夜间骑行安全性。", "category": "商品信息"},
        {"id": "product_lock_1", "text": "自行车锁 SecureLock Max,售价 199 元。特点:高强度U型锁,高碳钢材质,锁径13mm,长度230mm,防盗等级10级。配备钥匙和密码双重保护,安全可靠。", "category": "商品信息"},
        {"id": "faq_return_policy", "text": "退货政策:商品自签收之日起7天内,如存在质量问题或不满意,可申请退货。退货商品需保持原包装完整,未使用。退款将在收到退货后3-5个工作日内原路返回。非质量问题退货需承担运费。", "category": "常见问题"},
        {"id": "faq_shipping", "text": "配送时间:订单确认后,我们会在24小时内发货。市内配送1-2天,省内配送2-3天,省外配送3-5天。偏远地区可能需要额外1-2天。支持顺丰、圆通、中通等多家快递公司。", "category": "常见问题"},
        {"id": "faq_warranty", "text": "质保服务:所有自行车提供1年质保,配件提供6个月质保。质保期内非人为损坏可免费维修或更换。人为损坏、改装、事故等不在质保范围内。提供终身维护服务,质保期外维修仅收取成本费。", "category": "常见问题"},
        {"id": "faq_payment", "text": "支付方式:支持微信支付、支付宝、银联卡、信用卡等多种支付方式。支持货到付款(部分地区),但需额外支付10元手续费。支持分期付款,3期免息,6期、12期有手续费。", "category": "常见问题"},
        {"id": "doc_bike_installation", "text": "自行车安装教程:详细的图文教程和视频指导,教您如何正确组装自行车。包括车把安装、座椅调节、刹车调试、变速器校准等步骤。视频教程链接: https://example.com/bike-installation-tutorial", "category": "产品文档"},
        {"id": "doc_bike_maintenance", "text": "自行车保养指南:定期保养可延长自行车使用寿命。包括链条清洁上油、刹车片检查、轮胎气压检查、螺丝紧固等。建议每月检查一次,骑行500公里后进行深度保养。保养手册: https://example.com/bike-maintenance-guide", "category": "产品文档"},
        {"id": "doc_size_guide", "text": "尺寸选择指南:根据身高选择合适的自行车尺寸非常重要。身高150-165cm选择S码(14-16寸),165-175cm选择M码(17-18寸),175-185cm选择L码(19-20寸),185cm以上选择XL码(21-22寸)。详细指南: https://example.com/size-guide", "category": "产品文档"},
        {"id": "doc_safety_tips", "text": "骑行安全提示:骑行前检查刹车、轮胎、车灯等;佩戴头盔和反光衣;遵守交通规则;夜间骑行开启车灯;雨天路滑减速慢行;不要单手骑车或载人超重。安全骑行手册: https://example.com/safety-tips", "category": "产品文档"},
        {"id": "about_brand", "text": "我们是一家专注于高品质自行车的品牌,成立于2010年。我们的使命是让每个人都能享受骑行的乐趣。产品涵盖山地车、公路车、城市车等多个系列,畅销全国200多个城市。我们提供专业的售前咨询和售后服务。", "category": "品牌信息"},
        {"id": "customer_service", "text": "客服服务:在线客服工作时间 9:00-21:00,7天无休。支持电话咨询、在线聊天、邮件等多种方式。我们承诺:咨询响应时间不超过2分钟,问题解决不超过24小时。客服热线: 400-888-8888", "category": "客服信息"}
    ]
    
    print(f"\n📝 准备插入 {len(knowledge_data)} 条知识...")
    
    # 生成嵌入向量
    ids = []
    embeddings = []
    documents = []
    metadatas = []
    
    for i, item in enumerate(knowledge_data):
        print(f"   [{i+1}/{len(knowledge_data)}] {item['id']}")
        
        embedding = generate_embedding(item["text"])
        ids.append(item["id"])
        embeddings.append(embedding)
        documents.append(item["text"])
        metadatas.append({"category": item["category"]})
        
        time.sleep(0.3)  # 避免请求过快
    
    # 向 Chroma 添加文档
    print(f"\n📤 添加到 Chroma...")
    
    payload = {
        "ids": ids,
        "documents": documents,
        "embeddings": embeddings,
        "metadatas": metadatas
    }
    
    # 向 Chroma 添加文档
    print(f"\n📤 添加到 Chroma...")
    
    payload = {
        "ids": ids,
        "documents": documents,
        "embeddings": embeddings,
        "metadatas": metadatas
    }
    
    try:
        response = requests.post(
            f"{chroma_url}/api/v2/tenants/{tenant}/databases/{database}/collections/{collection_id}/add",
            json=payload,
            timeout=30
        )
        
        if response.status_code not in [200, 201]:
            print(f"❌ 添加失败 (状态码 {response.status_code})")
            print(f"   响应: {response.text[:200]}")
            return False
        
        print(f"✅ 成功添加 {len(ids)} 条文档")
        
    except Exception as e:
        print(f"❌ 添加异常: {e}")
        return False
    
    # 测试查询
    print(f"\n🧪 测试查询...")
    test_query = "自行车的安装教程"
    test_embedding = generate_embedding(test_query)
    
    try:
        query_payload = {
            "query_embeddings": [test_embedding],
            "n_results": 3
        }
        
        response = requests.post(
            f"{chroma_url}/api/v2/tenants/{tenant}/databases/{database}/collections/{collection_id}/query",
            json=query_payload,
            timeout=10
        )
        
        if response.status_code == 200:
            results = response.json()
            documents_list = results.get("documents", [[]])[0] if "documents" in results else []
            print(f"   查询: '{test_query}'")
            print(f"   结果数: {len(documents_list)}")
            if documents_list:
                print(f"   Top 1: {documents_list[0][:100]}...")
        else:
            print(f"❌ 查询失败 (状态码 {response.status_code})")
            
    except Exception as e:
        print(f"❌ 查询异常: {e}")
    
    print(f"\n🎉 知识库初始化完成!")
    return True


if __name__ == "__main__":
    try:
        success = init_knowledge_base()
        sys.exit(0 if success else 1)
    except Exception as e:
        print(f"\n❌ 异常: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
