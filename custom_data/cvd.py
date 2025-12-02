import requests
import pandas as pd
import plotly.graph_objects as go
from plotly.subplots import make_subplots

def fetch_klines(symbol="BTCUSDT", interval="15m", limit=500):
    base_url = "https://fapi.binance.com"
    endpoint = "/fapi/v1/klines"
    params = {
        "symbol": symbol,
        "interval": interval,
        "limit": limit
    }
    try:
        resp = requests.get(base_url + endpoint, params=params)
        data = resp.json()
        
        # 币安 K线数据结构:
        # [0:Time, 1:Open, 2:High, 3:Low, 4:Close, 5:Volume, ... 9:Taker Buy Volume, ...]
        df = pd.DataFrame(data, columns=[
            'open_time', 'open', 'high', 'low', 'close', 'volume', 
            'close_time', 'quote_asset_volume', 'trades', 
            'taker_buy_volume', 'taker_buy_quote_asset_volume', 'ignore'
        ])
        
        # 类型转换
        numeric_cols = ['open', 'high', 'low', 'close', 'volume', 'taker_buy_volume']
        df[numeric_cols] = df[numeric_cols].apply(pd.to_numeric, axis=1)
        df['open_time'] = pd.to_datetime(df['open_time'], unit='ms')
        
        return df
    except Exception as e:
        print(f"Error: {e}")
        return None

def calculate_cvd(df):
    """
    计算 Delta 和 CVD
    """
    # 1. 计算主动卖出量 (总成交量 - 主动买入量)
    df['taker_sell_volume'] = df['volume'] - df['taker_buy_volume']
    
    # 2. 计算 Delta (净买入量)
    df['delta'] = df['taker_buy_volume'] - df['taker_sell_volume']
    
    # 3. 计算 CVD (Delta 的累加)
    df['cvd'] = df['delta'].cumsum()
    
    return df

def plot_price_vs_cvd(df, symbol):
    """
    绘制 价格 vs CVD 的对比图 (用于识别背离)
    """
    # 创建双子图: 上面是价格，下面是 CVD
    fig = make_subplots(rows=2, cols=1, shared_xaxes=True, 
                        vertical_spacing=0.05, row_heights=[0.7, 0.3],
                        subplot_titles=(f"{symbol} 价格走势", "CVD (累积成交量差)"))

    # 1. 价格 K线图
    fig.add_trace(go.Candlestick(
        x=df['open_time'],
        open=df['open'], high=df['high'],
        low=df['low'], close=df['close'],
        name="Price"
    ), row=1, col=1)

    # 2. CVD 线图
    # 根据 CVD 的升降改变颜色 (绿色上升，红色下降)
    fig.add_trace(go.Scatter(
        x=df['open_time'], 
        y=df['cvd'],
        mode='lines',
        name='CVD',
        line=dict(color='yellow', width=2)
    ), row=2, col=1)

    fig.update_layout(
        template="plotly_dark",
        height=800,
        xaxis_rangeslider_visible=False
    )
    
    fig.show()

# --- 执行 ---
if __name__ == "__main__":
    symbol = "BTCUSDT"
    df = fetch_klines(symbol, interval="15m")
    
    if df is not None:
        df = calculate_cvd(df)
        
        # 打印最后几行数据验证
        print(df[['open_time', 'close', 'delta', 'cvd']].tail())
        
        plot_price_vs_cvd(df, symbol)