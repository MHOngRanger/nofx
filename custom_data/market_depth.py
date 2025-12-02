import requests
import pandas as pd
import plotly.express as px
import numpy as np

def fetch_order_book(symbol="BTCUSDT", limit=1000):
    """
    获取币安合约市场的深度数据
    limit: 默认获取 1000 档深度
    """
    base_url = "https://fapi.binance.com"
    endpoint = "/fapi/v1/depth"
    
    params = {
        "symbol": symbol,
        "limit": limit
    }
    
    try:
        response = requests.get(base_url + endpoint, params=params, timeout=10)
        response.raise_for_status()
        return response.json()
    except Exception as e:
        print(f"请求失败: {e}")
        return None

def process_depth_data(data, price_bin_size=20):
    """
    处理深度数据并进行聚合（Binning）
    price_bin_size: 聚合的价格区间，例如 20 代表将每 $20 范围内的订单合并
    """
    # 1. 提取买单 (Bids) 和 卖单 (Asks)
    # 格式: [Price, Quantity]
    bids = pd.DataFrame(data['bids'], columns=['price', 'quantity'], dtype=float)
    asks = pd.DataFrame(data['asks'], columns=['price', 'quantity'], dtype=float)
    
    # 2. 标记方向
    bids['side'] = 'Buy (Support)'
    asks['side'] = 'Sell (Resistance)'
    
    # 3. 价格分箱 (Binning) - 核心步骤
    # 将价格向下取整到最近的 bin_size 倍数
    # 例如: 价格 90123, bin=100 -> 90100
    bids['price_bin'] = (bids['price'] // price_bin_size) * price_bin_size
    asks['price_bin'] = (asks['price'] // price_bin_size) * price_bin_size
    
    # 4. 按分箱聚合数量
    # 计算每个价格区间的总挂单量 (USDT 价值更直观，所以这里我们计算 Quantity * Price)
    bids['value_usd'] = bids['price'] * bids['quantity']
    asks['value_usd'] = asks['price'] * asks['quantity']
    
    bids_grouped = bids.groupby('price_bin')['value_usd'].sum().reset_index()
    asks_grouped = asks.groupby('price_bin')['value_usd'].sum().reset_index()
    
    # 重新标记 side，方便绘图
    bids_grouped['side'] = 'Buy (Support)'
    asks_grouped['side'] = 'Sell (Resistance)'
    
    # 为了让买单条形图向左显示（如果需要漏斗图效果），这里不做负值处理，直接用颜色区分
    return pd.concat([bids_grouped, asks_grouped])

def plot_market_depth(df, symbol, current_price):
    """
    绘制挂单密度图
    """
    if df is None or df.empty:
        print("无数据绘图")
        return

    # 筛选范围：为了图表好看，只显示当前价格上下 2% 范围内的挂单
    lower_bound = current_price * 0.98
    upper_bound = current_price * 1.02
    
    df_filtered = df[(df['price_bin'] >= lower_bound) & (df['price_bin'] <= upper_bound)]

    # 使用 Plotly 绘制条形图
    fig = px.bar(
        df_filtered,
        y="price_bin",      # Y轴显示价格
        x="value_usd",      # X轴显示挂单金额
        color="side",       # 颜色区分买卖
        orientation='h',    # 水平方向
        title=f"{symbol} 挂单深度分布 (聚合精度: $20)",
        labels={'price_bin': '价格 (Price)', 'value_usd': '挂单总价值 (USD)', 'side': '方向'},
        color_discrete_map={'Buy (Support)': '#00b15d', 'Sell (Resistance)': '#ff5e5e'} # 经典的红绿配色
    )

    # 添加当前价格线
    fig.add_hline(y=current_price, line_dash="dash", line_color="white", annotation_text="当前价格")

    fig.update_layout(
        template="plotly_dark",
        bargap=0.1,         # 柱子之间的间隙
        height=700,
        xaxis_title="挂单堆积量 (USD Value)"
    )
    
    fig.show()

# --- 主程序 ---
if __name__ == "__main__":
    symbol = "BTCUSDT"
    print(f"正在获取 {symbol} 深度数据...")
    
    depth_data = fetch_order_book(symbol)
    
    if depth_data:
        # 获取当前盘口中间价，用于定位图表中心
        best_bid = float(depth_data['bids'][0][0])
        best_ask = float(depth_data['asks'][0][0])
        mid_price = (best_bid + best_ask) / 2
        
        print(f"当前价格约为: ${mid_price:,.2f}")
        
        # 处理数据 (按 $20 为一档进行聚合)
        df_depth = process_depth_data(depth_data, price_bin_size=20)
        
        # 绘图
        plot_market_depth(df_depth, symbol, mid_price)