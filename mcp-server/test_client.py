"""
MCP Client 测试脚本
用于测试 MCP Server 的工具调用
"""
import asyncio
import json
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


async def test_mcp_tools():
    """测试 MCP 工具调用"""
    
    # 配置服务器参数
    server_params = StdioServerParameters(
        command="python",
        args=["server.py"]
    )
    
    print("🔌 连接到 MCP Server...")
    
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            # 初始化会话
            await session.initialize()
            print("✅ MCP Session 初始化成功\n")
            
            # 列出可用工具
            tools_result = await session.list_tools()
            print(f"📋 可用工具 ({len(tools_result.tools)}):")
            for tool in tools_result.tools:
                print(f"   - {tool.name}: {tool.description}")
            print()
            
            # 测试 1: 创建订单
            print("🧪 测试 1: 创建订单")
            create_result = await session.call_tool(
                "create_order",
                arguments={
                    "productId": 1,
                    "quantity": 2,
                    "customerName": "鹿城",
                    "customerPhone": "13800138000",
                    "shippingAddress": "北京市朝阳区建国路1号"
                }
            )
            print("结果:")
            for content in create_result.content:
                print(content.text)
            print()
            
            # 测试 2: 查询所有订单
            print("🧪 测试 2: 查询所有订单")
            query_all_result = await session.call_tool(
                "query_order",
                arguments={}
            )
            print("结果:")
            for content in query_all_result.content:
                print(content.text)
            print()
            
            # 测试 3: 查询指定订单（需要从上面的结果中提取订单号）
            # 这里假设订单号格式为 ORD-xxxxx
            # print("🧪 测试 3: 查询指定订单")
            # query_one_result = await session.call_tool(
            #     "query_order",
            #     arguments={"orderNumber": "ORD-12345"}
            # )
            # print("结果:")
            # for content in query_one_result.content:
            #     print(content.text)
            # print()


if __name__ == "__main__":
    asyncio.run(test_mcp_tools())
