import requests
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
from datetime import datetime

def fetch_liquidation_data(symbol="BTCUSDT", limit=1000):
    """
    从币安 U本位合约 API 获取最近的强平订单数据
    注意：中国大陆地区可能需要配置代理或使用 VPN
    """
    base_url = "https://fapi.binance.com"
    endpoint = "/fapi/v1/forceOrders"
    
    params = {
        "symbol": symbol,
        "limit": limit
    }
    
    try:
        response = requests.get(base_url + endpoint, params=params, timeout=10)
        response.raise_for_status()
        data = response.json()
        
        if not data:
            print("未获取到数据，请检查网络或交易对。")
            return None
            
        return data
    except Exception as e:
        print(f"请求失败: {e}")
        return None

def process_data(data):
    """
    将 API 返回的 JSON 数据转换为 DataFrame 并进行处理
    """
    df = pd.DataFrame(data)
    
    # 数据转换
    df['price'] = df['price'].astype(float)
    df['origQty'] = df['origQty'].astype(float) # 原始数量
    df['time'] = pd.to_datetime(df['time'], unit='ms')
    
    # 计算名义价值 (USD Value) = 价格 * 数量
    df['value_usd'] = df['price'] * df['origQty']
    
    # 标记清算方向
    # SELL side means a Long position was liquidated (selling to close)
    # BUY side means a Short position was liquidated (buying to close)
    df['liquidation_type'] = df['side'].apply(lambda x: '多头清算 (Longs Rekt)' if x == 'SELL' else '空头清算 (Shorts Rekt)')
    
    return df

def plot_liquidation_heatmap(df, symbol):
    """
    使用 Plotly 绘制清算密度热力图
    """
    if df is None or df.empty:
        print("没有数据用于绘图")
        return

    # 创建密度热力图
    # X轴: 时间, Y轴: 价格, 颜色深浅: 清算金额 (USD)
    fig = px.density_heatmap(
        df, 
        x="time", 
        y="price", 
        z="value_usd", 
        histfunc="sum",
        nbinsx=30, # X轴的时间切片数量，可调整
        nbinsy=30, # Y轴的价格切片数量，可调整
        color_continuous_scale="Viridis", # 颜色主题
        title=f"{symbol} 强平热力图 (基于最近 {len(df)} 笔强平订单)",
        labels={'value_usd': '清算总金额 (USD)', 'time': '时间', 'price': '价格'}
    )

    # 添加散点图层，显示具体的清算点
    # 不同的形状/颜色代表多头或空头清算
    fig.add_trace(
        go.Scatter(
            x=df['time'], 
            y=df['price'], 
            mode='markers',
            marker=dict(
                size=df['value_usd'] / df['value_usd'].max() * 20, # 气泡大小根据金额动态调整
                opacity=0.6,
                color=df['side'].map({'SELL': 'red', 'BUY': 'green'}), # 红色代表多头死(卖出)，绿色代表空头死(买入)
                line=dict(width=1, color='DarkSlateGrey')
            ),
            text=df['liquidation_type'] + "<br>金额: $" + df['value_usd'].round(0).astype(str),
            name='具体清算订单'
        )
    )

    fig.update_layout(
        template="plotly_dark",
        hovermode="closest",
        height=600
    )
    
    fig.show()

# --- 主程序执行 ---
if __name__ == "__main__":
    symbol = "BTCUSDT"
    print(f"正在获取 {symbol} 的数据...")
    
    raw_data = fetch_liquidation_data(symbol)
    
    if raw_data:
        df = process_data(raw_data)
        
        # 简单的统计输出
        total_vol = df['value_usd'].sum()
        print(f"--- 统计摘要 ---")
        print(f"获取记录数: {len(df)}")
        print(f"时间范围: {df['time'].min()} 至 {df['time'].max()}")
        print(f"累计清算金额: ${total_vol:,.2f}")
        
        plot_liquidation_heatmap(df, symbol)