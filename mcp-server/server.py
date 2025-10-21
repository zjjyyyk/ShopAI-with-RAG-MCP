"""
MCP Server - 提供订单管理工具服务
使用 FastMCP 实现标准 MCP 协议
"""
import os
import requests
from mcp.server.fastmcp import FastMCP

# 创建 MCP 服务器
mcp = FastMCP("OrderManager")

# Java Shop API 地址
JAVA_SHOP_URL = os.getenv("JAVA_SHOP_URL", "http://java-shop:8080")


@mcp.tool()
def search_product(keyword: str) -> str:
    """
    搜索商品
    
    Args:
        keyword: 商品名称关键词
    
    Returns:
        匹配的商品列表
    """
    try:
        url = f"{JAVA_SHOP_URL}/api/products/search?keyword={keyword}"
        response = requests.get(url, timeout=10)
        
        if response.status_code != 200:
            return f"❌ 搜索商品失败：HTTP {response.status_code}"
        
        products = response.json()
        
        if not products:
            return f"❌ 未找到与 '{keyword}' 相关的商品"
        
        result = f"🔍 找到 {len(products)} 个商品：\n\n"
        for product in products:
            result += f"商品ID：{product.get('id')}\n"
            result += f"商品名称：{product.get('name')}\n"
            result += f"价格：¥{product.get('price')}\n"
            result += f"类别：{product.get('category')}\n"
            result += f"库存：{product.get('stock')}\n"
            result += f"描述：{product.get('description')}\n"
            result += "---\n"
        
        return result
        
    except requests.exceptions.RequestException as e:
        return f"❌ 搜索商品失败：{str(e)}"
    except Exception as e:
        return f"❌ 系统错误：{str(e)}"


@mcp.tool()
def create_order(
    productName: str,
    quantity: int,
    customerName: str,
    customerPhone: str,
    shippingAddress: str
) -> str:
    """
    创建新订单
    
    Args:
        productName: 商品名称（如"山地自行车"、"公路车"等）
        quantity: 购买数量
        customerName: 客户姓名
        customerPhone: 客户电话
        shippingAddress: 收货地址
    
    Returns:
        订单创建结果（包含订单号）
    """
    try:
        # 1. 先搜索商品，获取商品ID
        search_url = f"{JAVA_SHOP_URL}/api/products/search?keyword={productName}"
        search_response = requests.get(search_url, timeout=10)
        
        if search_response.status_code != 200:
            return f"❌ 搜索商品失败：HTTP {search_response.status_code}"
        
        products = search_response.json()
        
        if not products:
            return f"❌ 未找到商品 '{productName}'，请检查商品名称是否正确"
        
        # 使用第一个匹配的商品
        product = products[0]
        product_id = product.get('id')
        product_name = product.get('name')
        product_price = product.get('price')
        
        # 2. 创建订单
        url = f"{JAVA_SHOP_URL}/api/orders"
        payload = {
            "productId": product_id,
            "quantity": quantity,
            "customerName": customerName,
            "customerPhone": customerPhone,
            "shippingAddress": shippingAddress
        }
        
        response = requests.post(url, json=payload, timeout=10)
        
        if response.status_code == 200:
            order = response.json()
            total_price = product_price * quantity
            return f"""✅ 订单创建成功！

订单号：{order.get('orderNumber')}
商品名称：{product_name}
单价：¥{product_price}
数量：{order.get('quantity')}
总价：¥{total_price}
客户姓名：{order.get('customerName')}
联系电话：{order.get('customerPhone')}
收货地址：{order.get('shippingAddress')}
订单状态：{order.get('status')}

您可以随时查询订单状态或取消订单。"""
        else:
            return f"❌ 创建订单失败：HTTP {response.status_code}"
            
    except requests.exceptions.RequestException as e:
        return f"❌ 创建订单失败：{str(e)}"
    except Exception as e:
        return f"❌ 系统错误：{str(e)}"


@mcp.tool()
def query_order(orderNumber: str = None) -> str:
    """
    查询订单信息
    
    Args:
        orderNumber: 订单号（可选，如果不提供则返回所有订单）
    
    Returns:
        订单信息
    """
    try:
        url = f"{JAVA_SHOP_URL}/api/orders"
        response = requests.get(url, timeout=10)
        
        if response.status_code != 200:
            return f"❌ 查询订单失败：HTTP {response.status_code}"
        
        orders = response.json()
        
        if not orders:
            return "📋 暂无订单记录"
        
        # 如果指定了订单号，只返回该订单
        if orderNumber:
            target_order = None
            for order in orders:
                if order.get('orderNumber') == orderNumber:
                    target_order = order
                    break
            
            if not target_order:
                return f"❌ 未找到订单：{orderNumber}"
            
            return f"""📋 订单详情

订单号：{target_order.get('orderNumber')}
商品ID：{target_order.get('productId')}
数量：{target_order.get('quantity')}
客户姓名：{target_order.get('customerName')}
联系电话：{target_order.get('customerPhone')}
收货地址：{target_order.get('shippingAddress')}
订单状态：{target_order.get('status')}"""
        
        # 返回所有订单
        result = f"📋 共有 {len(orders)} 个订单：\n\n"
        for order in orders:
            result += f"订单号：{order.get('orderNumber')}\n"
            result += f"商品ID：{order.get('productId')}\n"
            result += f"数量：{order.get('quantity')}\n"
            result += f"客户：{order.get('customerName')}\n"
            result += f"状态：{order.get('status')}\n"
            result += "---\n"
        
        return result
        
    except requests.exceptions.RequestException as e:
        return f"❌ 查询订单失败：{str(e)}"
    except Exception as e:
        return f"❌ 系统错误：{str(e)}"


@mcp.tool()
def cancel_order(orderNumber: str) -> str:
    """
    取消订单
    
    Args:
        orderNumber: 订单号
    
    Returns:
        取消结果
    """
    try:
        url = f"{JAVA_SHOP_URL}/api/orders/{orderNumber}"
        response = requests.delete(url, timeout=10)
        
        if response.status_code == 200:
            return f"✅ 订单 {orderNumber} 已成功取消"
        elif response.status_code == 404:
            return f"❌ 订单 {orderNumber} 不存在"
        else:
            return f"❌ 取消订单失败：HTTP {response.status_code}"
            
    except requests.exceptions.RequestException as e:
        return f"❌ 取消订单失败：{str(e)}"
    except Exception as e:
        return f"❌ 系统错误：{str(e)}"


if __name__ == "__main__":
    # 使用 stdio 传输协议启动服务器
    mcp.run(transport='stdio')
