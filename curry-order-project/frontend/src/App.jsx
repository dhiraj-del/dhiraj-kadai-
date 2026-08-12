import React, { useState, useEffect } from 'react';

const API_BASE = window.location.origin.includes('-3000.')
  ? window.location.origin.replace('-3000.', '-8080.') // Codespaces環境用
  : `${window.location.protocol}//${window.location.hostname}:8080`; // AWS / ローカル環境用

function App() {
  const [menuItems, setMenuItems] = useState([]);
  const [cart, setCart] = useState([]);
  const [tableNo, setTableNo] = useState('1');
  const [spicinessMap, setSpicinessMap] = useState({});
  const [activeTab, setActiveTab] = useState('menu'); // 'menu' | 'cart' | 'history'
  const [orderHistory, setOrderHistory] = useState([]);
  const [checkoutMessage, setCheckoutMessage] = useState('');

  const fetchOptions = (options = {}) => {
    return {
      ...options,
      credentials: 'include',
    };
  };

  useEffect(() => {
    fetchMenu();
  }, []);

  useEffect(() => {
    if (activeTab === 'history') {
      fetchHistory();
    }
  }, [activeTab, tableNo]);

  const fetchMenu = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/menu`, fetchOptions());
      if (res.ok) {
        const data = await res.json();
        setMenuItems(data || []);
      }
    } catch (e) {
      console.error("メニュー取得エラー:", e);
    }
  };

  const fetchHistory = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/orders/history?table_no=${tableNo}`, fetchOptions());
      if (res.ok) {
        const data = await res.json();
        setOrderHistory(data || []);
      }
    } catch (e) {
      console.error("履歴取得エラー:", e);
    }
  };

  const addToCart = (item) => {
    const spiciness = spicinessMap[item.id] || (item.category === 'Curry' ? '普通 (Medium)' : '-');
    const existingIndex = cart.findIndex(c => c.id === item.id && c.spiciness === spiciness);

    if (existingIndex > -1) {
      const updated = [...cart];
      updated[existingIndex].quantity += 1;
      setCart(updated);
    } else {
      setCart([...cart, { ...item, quantity: 1, spiciness }]);
    }
  };

  const updateQuantity = (index, delta) => {
    const updated = [...cart];
    updated[index].quantity += delta;
    if (updated[index].quantity <= 0) {
      updated.splice(index, 1);
    }
    setCart(updated);
  };

  const handleOrderSubmit = async () => {
    if (cart.length === 0) return;

    const payload = {
      table_no: tableNo,
      items: cart.map(item => ({
        menu_item_id: item.id,
        quantity: item.quantity,
        spiciness: item.spiciness,
      })),
    };

    try {
      const res = await fetch(`${API_BASE}/api/orders`, fetchOptions({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }));

      if (res.ok) {
        alert('厨房へ注文を送信しました！');
        setCart([]);
        setActiveTab('history');
      } else {
        alert('注文に失敗しました。');
      }
    } catch (e) {
      console.error(e);
      alert('通信エラーが発生しました。');
    }
  };

  const handleCheckout = async () => {
    if (!window.confirm(`卓番号 [${tableNo}] のお会計を行いますか？`)) return;

    try {
      const res = await fetch(`${API_BASE}/api/checkout`, fetchOptions({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ table_no: tableNo }),
      }));

      const data = await res.json();
      if (res.ok) {
        setCheckoutMessage(`${data.message} 合計金額: ¥${data.grand_total.toLocaleString()}`);
        setOrderHistory([]);
      } else {
        alert(data.message || '精算処理に失敗しました。');
      }
    } catch (e) {
      console.error(e);
      alert('通信エラーが発生しました。');
    }
  };

  const cartTotal = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
  const cartItemCount = cart.reduce((sum, item) => sum + item.quantity, 0);

  return (
    <div className="app-container">
      <header className="navbar">
        <div className="nav-content">
          <h1 className="logo" onClick={() => setActiveTab('menu')}>🇳🇵 ナマステ カレーハウス</h1>
          <div className="table-selector">
            <label>卓番号: </label>
            <select value={tableNo} onChange={(e) => setTableNo(e.target.value)}>
              <option value="1">1番テーブル</option>
              <option value="2">2番テーブル</option>
              <option value="3">3番テーブル</option>
            </select>
          </div>
        </div>
      </header>

      <nav className="sub-nav">
        <button className={activeTab === 'menu' ? 'active' : ''} onClick={() => setActiveTab('menu')}>
          メニュー一覧
        </button>
        <button className={activeTab === 'cart' ? 'active' : ''} onClick={() => setActiveTab('cart')}>
          注文確認 ({cartItemCount})
        </button>
        <button className={activeTab === 'history' ? 'active' : ''} onClick={() => setActiveTab('history')}>
          注文履歴・精算
        </button>
      </nav>

      <main className="main-layout">
        {checkoutMessage && (
          <div className="checkout-banner">
            <span>{checkoutMessage}</span>
            <button onClick={() => setCheckoutMessage('')}>閉じる</button>
          </div>
        )}

        {/* メニュー一覧画面 */}
        {activeTab === 'menu' && (
          <div className="menu-grid">
            {menuItems.map(item => (
              <div key={item.id} className="card menu-card">
                <img src={item.image_url} alt={item.name} className="menu-img" />
                <div className="menu-info">
                  <span className="badge">{item.category}</span>
                  <h3>{item.name}</h3>
                  <p className="description">{item.description}</p>
                  <p className="price">¥{item.price.toLocaleString()}</p>
                  
                  {item.category === 'Curry' && (
                    <div className="spiciness-select">
                      <label>辛さ: </label>
                      <select 
                        value={spicinessMap[item.id] || '普通 (Medium)'}
                        onChange={(e) => setSpicinessMap({ ...spicinessMap, [item.id]: e.target.value })}
                      >
                        <option value="甘口 (Mild)">甘口 (Mild)</option>
                        <option value="普通 (Medium)">普通 (Medium)</option>
                        <option value="中辛 (Hot)">中辛 (Hot)</option>
                        <option value="激辛 (Very Hot)">激辛 (Very Hot)</option>
                      </select>
                    </div>
                  )}

                  <button className="btn-primary btn-block" onClick={() => addToCart(item)}>
                    カートに追加
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* カート確認画面 */}
        {activeTab === 'cart' && (
          <div className="card cart-container">
            <h2>カート内の確認 (卓番号: {tableNo})</h2>
            {cart.length === 0 ? (
              <p className="empty-message">カートに商品が入っていません。</p>
            ) : (
              <>
                <div className="cart-list">
                  {cart.map((item, index) => (
                    <div key={index} className="cart-item">
                      <div>
                        <h4>{item.name}</h4>
                        {item.spiciness !== '-' && <span className="spiciness-tag">辛さ: {item.spiciness}</span>}
                        <p className="price">¥{item.price.toLocaleString()} × {item.quantity}</p>
                      </div>
                      <div className="quantity-controls">
                        <button onClick={() => updateQuantity(index, -1)}>-</button>
                        <span>{item.quantity}</span>
                        <button onClick={() => updateQuantity(index, 1)}>+</button>
                      </div>
                    </div>
                  ))}
                </div>
                <div className="cart-summary">
                  <h3>合計金額: ¥{cartTotal.toLocaleString()}</h3>
                  <button className="btn-primary btn-block" onClick={handleOrderSubmit}>
                    厨房へ注文を確定する
                  </button>
                </div>
              </>
            )}
          </div>
        )}

        {/* 注文履歴・精算画面 */}
        {activeTab === 'history' && (
          <div className="history-container">
            <h2>現在の注文状況 (卓番号: {tableNo})</h2>
            {orderHistory.length === 0 ? (
              <p className="empty-message">未精算の注文履歴はありません。</p>
            ) : (
              <>
                {orderHistory.map(order => (
                  <div key={order.id} className="card history-card">
                    <div className="history-header">
                      <span>注文ID: #{order.id}</span>
                      <span className="time">{new Date(order.created_at).toLocaleTimeString()}</span>
                    </div>
                    <div className="history-items">
                      {order.items.map((item, idx) => (
                        <div key={idx} className="history-item">
                          <span>{item.name} {item.spiciness !== '-' ? `(${item.spiciness})` : ''} × {item.quantity}</span>
                          <span>¥{(item.price * item.quantity).toLocaleString()}</span>
                        </div>
                      ))}
                    </div>
                    <div className="history-total">
                      小計: ¥{order.total.toLocaleString()}
                    </div>
                  </div>
                ))}

                <div className="card checkout-card">
                  <h3>合計請求額: ¥{orderHistory.reduce((sum, o) => sum + o.total, 0).toLocaleString()}</h3>
                  <button className="btn-danger btn-block" onClick={handleCheckout}>
                    お会計（精算処理）を実行
                  </button>
                </div>
              </>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;