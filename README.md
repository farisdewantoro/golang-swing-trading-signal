# Swing Trading Signal API

Sistem analisis teknikal saham Indonesia menggunakan Gemini AI API dengan data OHLC dari Yahoo Finance API untuk memberikan sinyal trading swing dengan analisis teknikal yang mendalam.

## 🚀 Features

- **Enhanced Individual Stock Analysis**: Analisis teknikal saham individual dengan 15+ indikator dan analisis mendalam
- **Advanced Position Monitoring**: Monitoring posisi trading dengan analisis teknikal komprehensif
- **AI-Powered Technical Analysis**: Analisis teknikal mendalam menggunakan Gemini AI
- **Multiple Timeframe Analysis**: Analisis trend short-term dan medium-term
- **Advanced Technical Indicators**: EMA, RSI, MACD, Stochastic, Bollinger Bands, dan lebih banyak lagi
- **Comprehensive Risk Analysis**: Analisis risk-reward dengan multiple probability assessments
- **Detailed Support/Resistance**: Identifikasi key levels dan psychological levels
- **Volume Analysis**: Analisis volume trend dan confirmation
- **Exit Strategy Recommendations**: Rekomendasi exit yang detail dengan reasoning
- **Technical Summary**: Ringkasan teknikal dengan key insights
- **OHLCV Analysis**: Analisis detail pergerakan harga harian
- **Telegram Bot Integration**: Analisis saham dan notifikasi monitoring posisi via Telegram dengan format yang enhanced

## 🛠️ Tech Stack

- **Language**: Golang 1.22
- **Web Framework**: Gin
- **Configuration**: Viper
- **Logging**: Logrus
- **Data Source**: Yahoo Finance API
- **AI Integration**: Gemini API
- **Telegram Bot**: Telebot v3

## 📋 Prerequisites

- Go 1.22 or higher
- Gemini AI API Key
- Telegram Bot Token (optional, for Telegram integration)
- Internet connection for Yahoo Finance API

## 🚀 Quick Start

### 1. Clone Repository
```bash
git clone <repository-url>
cd golang-swing-trading-signal
```

### 2. Install Dependencies
```bash
go mod tidy
```

### 3. Setup Environment
```bash
cp env.example .env
```

Edit `.env` file and add your API keys:
```env
# Server Configuration
PORT=8080
ENV=development

# Yahoo Finance API
YAHOO_FINANCE_BASE_URL=https://query1.finance.yahoo.com/v8/finance/chart

# Gemini AI API
GEMINI_API_KEY=your_gemini_api_key_here
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta/models
GEMINI_MODEL=gemini-pro

# Trading Configuration
DEFAULT_MAX_HOLDING_PERIOD_DAYS=5
CONFIDENCE_THRESHOLD=70

# Telegram Bot Configuration (Optional)
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
TELEGRAM_CHAT_ID=your_telegram_chat_id_here
TELEGRAM_WEBHOOK_URL=https://your-domain.com/telegram/webhook
```

### 4. Setup Telegram Bot (Optional)

Untuk menggunakan fitur Telegram bot:

1. Buat bot Telegram melalui @BotFather
2. Dapatkan bot token dan chat ID
3. Tambahkan ke file `.env`
4. Lihat [TELEGRAM_BOT_SETUP.md](TELEGRAM_BOT_SETUP.md) untuk panduan lengkap

### 5. Run Application
```bash
go run cmd/server/main.go
```

Server will start on `http://localhost:8080`

## 🤖 Telegram Bot Features

### Stock Analysis via Telegram
- Kirim kode saham langsung ke bot (contoh: `BBCA`, `ANTM`, `TLKM`)
- Gunakan command `/analyze <symbol>` untuk analisis detail
- Dapatkan analisis komprehensif termasuk:
  - Data harga terkini (OHLCV)
  - Analisis teknikal (EMA, RSI, MACD, Support/Resistance)
  - Rekomendasi trading
  - Analisis risk/reward
  - Level confidence

### Position Monitoring Notifications
- Notifikasi otomatis saat posisi dimonitor via API
- Update real-time performa posisi
- Penilaian risiko dan rekomendasi exit
- Tracking unrealized P&L

### Webhook Implementation
- **Real-time updates**: Tidak ada delay polling
- **Better performance**: Beban server lebih rendah
- **Production ready**: Cocok untuk aplikasi high-traffic
- **Automatic webhook management**: Setup dan cleanup otomatis

### Bot Commands
- `/start` - Pesan selamat datang
- `/help` - Bantuan dan contoh penggunaan
- `/analyze <symbol>` - Analisis saham tertentu

### Quick Webhook Setup
Untuk development lokal dengan ngrok:

```bash
# 1. Start server
go run cmd/server/main.go

# 2. Setup webhook (di terminal lain)
./scripts/setup_webhook.sh
```

## 📚 API Documentation

### Health Check
```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-13T09:00:00Z",
  "service": "swing-trading-signal"
}
```

### Telegram Bot Health Check
```bash
curl http://localhost:8080/api/v1/telegram/health
```

### Individual Stock Analysis
```bash
curl "http://localhost:8080/api/v1/analyze?symbol=BBCA"
```

**Parameters:**
- `symbol` (required): Kode saham Indonesia (contoh: ANTM, BBCA, TLKM)

**Response:**
```json
{
  "symbol": "ANTM",
  "analysis_date": "2025-01-13T09:00:00+07:00",
  "signal": "BUY",
  "ohlcv_analysis": {
    "open": 2740,
    "high": 2750,
    "low": 2730,
    "close": 2745,
    "volume": 80000,
    "explanation": "Harga pembukaan sesi pertama berada di 2740..."
  },
  "technical_analysis": {
    "trend": "BULLISH",
    "ema_9": 8750,
    "ema_21": 8700,
    "ema_signal": "BULLISH",
    "rsi": 65.5,
    "rsi_signal": "NEUTRAL",
    "macd_signal": "BULLISH",
    "support_level": 8500,
    "resistance_level": 9200,
    "volume_trend": "HIGH",
    "candlestick_pattern": "BULLISH"
  },
  "recommendation": {
    "action": "BUY",
    "buy_price": 8750,
    "target_price": 9200,
    "cut_loss": 8400,
    "confidence_level": 85,
    "reasoning": "Analisis teknikal menunjukkan momentum bullish...",
    "risk_reward_analysis": {
      "potential_profit": 450,
      "potential_profit_percentage": 5.14,
      "potential_loss": 350,
      "potential_loss_percentage": 4.0,
      "risk_reward_ratio": 1.29,
      "risk_level": "LOW",
      "expected_holding_period": "3-5 days",
      "success_probability": 75
    }
  },
  "risk_level": "LOW"
}
```

### Position Monitoring
```bash
curl -X POST http://localhost:8080/api/v1/monitor-position \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BBCA",
    "buy_price": 9000,
    "buy_time": "2025-01-13T09:00:00+07:00",
    "max_holding_period_days": 5
  }'
```

**Request Body:**
```json
{
  "symbol": "BBCA",
  "buy_price": 9000,
  "buy_time": "2025-01-13T09:00:00+07:00",
  "max_holding_period_days": 5
}
```

**Note:** Jika Telegram bot dikonfigurasi, notifikasi akan dikirim otomatis ke chat Telegram.

**Response:**
```json
{
  "symbol": "BBCA",
  "current_price": 9100,
  "position_age_days": 2,
  "max_holding_period_days": 5,
  "ohlcv_analysis": {
    "open": 9050,
    "high": 9120,
    "low": 9030,
    "close": 9100,
    "volume": 120000,
    "explanation": "Harga pembukaan sesi pertama berada di 9050..."
  },
  "recommendation": {
    "action": "HOLD",
    "reasoning": "Saham masih dalam momentum bullish...",
    "technical_analysis": {
      "trend": "BULLISH",
      "ema_signal": "BULLISH",
      "rsi": 68.5,
      "rsi_signal": "NEUTRAL",
      "macd_signal": "BULLISH",
      "support_level": 8950,
      "resistance_level": 9200,
      "volume_trend": "HIGH"
    },
    "risk_reward_analysis": {
      "current_profit": 100,
      "current_profit_percentage": 1.11,
      "remaining_potential_profit": 400,
      "remaining_potential_profit_percentage": 4.44,
      "current_risk": 150,
      "current_risk_percentage": 1.67,
      "risk_reward_ratio": 2.67,
      "risk_level": "LOW",
      "days_remaining": 3,
      "success_probability": 85,
      "exit_recommendation": {
        "target_exit_price": 9500,
        "stop_loss_price": 8950,
        "time_based_exit": "2025-01-16T09:00:00+07:00"
      }
    }
  },
  "position_metrics": {
    "unrealized_pnl": 100,
    "unrealized_pnl_percentage": 1.11,
    "days_remaining": 3,
    "risk_assessment": "LOW"
  },
  "next_review": "2025-01-16T09:00:00+07:00"
}
```

### Contoh Penggunaan Lainnya

#### Analisis Saham Lainnya
```bash
# Analisis saham ANTM
curl "http://localhost:8080/api/v1/analyze?symbol=ANTM"

# Analisis saham TLKM
curl "http://localhost:8080/api/v1/analyze?symbol=TLKM"

# Analisis saham ASII
curl "http://localhost:8080/api/v1/analyze?symbol=ASII"
```

#### Monitoring Posisi dengan Data Berbeda
```bash
# Monitoring posisi BBCA dengan data berbeda
curl -X POST http://localhost:8080/api/v1/monitor-position \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BBCA",
    "buy_price": 9500,
    "buy_time": "2025-01-10T09:00:00+07:00",
    "max_holding_period_days": 7
  }'

# Monitoring posisi ANTM
curl -X POST http://localhost:8080/api/v1/monitor-position \
  -H "Content-Type: application/json" \
  -d '{
  "symbol": "RAJA",
  "buy_price": 2830.00,
  "buy_time": "2025-06-13T09:14:00+07:00",
  "max_holding_period_days": 5
}'
```

#### Testing dengan jq (untuk formatting JSON)
```bash
# Analisis dengan output yang diformat
curl "http://localhost:8080/api/v1/analyze?symbol=BBCA" | jq '.'

# Filter hanya signal dan recommendation
curl "http://localhost:8080/api/v1/analyze?symbol=BBCA" | jq '{symbol, signal, recommendation}'

# Filter hanya risk reward analysis
curl "http://localhost:8080/api/v1/analyze?symbol=BBCA" | jq '.recommendation.risk_reward_analysis'
```

## 🔧 Technical Indicators

Sistem menggunakan 15+ indikator teknikal canggih dengan analisis mendalam:

### Core Technical Indicators
1. **EMA (Exponential Moving Average)**: EMA 9 & EMA 21 dengan signal analysis
2. **RSI (Relative Strength Index)**: 14 periode dengan overbought/oversold signals
3. **MACD**: Moving Average Convergence Divergence (12,26,9)
4. **Stochastic Oscillator**: 14,3,3 dengan signal analysis
5. **Bollinger Bands**: Position analysis (UPPER/MIDDLE/LOWER)
6. **Support & Resistance**: Level support dan resistance dengan key levels
7. **Volume Analysis**: Volume trend dan confirmation analysis
8. **Candlestick Patterns**: Pola candlestick harian

### Enhanced Analysis Features
9. **Multiple Timeframe Analysis**: Short-term (1-3 days) dan medium-term (1-2 weeks) trends
10. **Trend Strength Assessment**: STRONG/MODERATE/WEAK trend evaluation
11. **Market Structure Analysis**: UPTREND/DOWNTREND/SIDEWAYS identification
12. **Breakout Potential**: HIGH/MEDIUM/LOW breakout probability
13. **Momentum Indicators**: Comprehensive momentum analysis
14. **Technical Score**: 0-100 scoring system based on all indicators
15. **Volume Confirmation**: Price-volume relationship analysis
16. **Max Holding Period Days**: Intelligent recommendations for optimal holding periods (1-7 days) based on technical analysis

### Advanced Risk Analysis
17. **Multiple Probability Assessments**: Success, trend, volume, dan technical probabilities
18. **Enhanced Exit Recommendations**: Target price, stop loss, time-based exit dengan reasoning
19. **Position Health Assessment**: Position health, trend alignment, volume support
20. **Technical Summary**: Overall signal, key insights, dan confidence levels

## 🤖 Enhanced Telegram Bot Features

### Enhanced Stock Analysis via Telegram
- **Comprehensive Technical Analysis**: 15+ technical indicators dengan analisis mendalam
- **Multiple Timeframe Trends**: Short-term dan medium-term trend analysis
- **Advanced Risk Assessment**: Multiple probability assessments
- **Detailed Exit Strategies**: Exit recommendations dengan reasoning dan conditions
- **Technical Summary**: Key insights dan confidence levels
- **Max Holding Period Days**: Optimal holding period recommendations (1-7 days) displayed prominently in messages

### Enhanced Position Monitoring via Telegram
- **Real-time Technical Analysis**: Live technical analysis untuk posisi yang sedang berjalan
- **Position Health Monitoring**: Assessment kesehatan posisi dan trend alignment
- **Advanced Exit Recommendations**: Detailed exit strategies dengan multiple conditions
- **Risk Probability Analysis**: Trend, volume, dan technical probabilities
- **Technical Summary**: Comprehensive summary dengan key insights

### Enhanced Message Format
Telegram bot sekarang menampilkan:
- **Trend Analysis**: Overall, short-term, dan medium-term trends
- **Technical Indicators**: EMA, RSI, MACD, Stochastic, Bollinger Bands
- **Key Levels**: Support, resistance, dan key psychological levels
- **Volume Analysis**: Volume trend dan confirmation
- **Risk Analysis**: Multiple probability assessments
- **Exit Recommendations**: Detailed exit strategies
- **Technical Summary**: Key insights dan confidence levels

## 📊 Enhanced Data Flow

1. **Data Collection**: Ambil data OHLC dari Yahoo Finance API
2. **Enhanced AI Analysis**: Kirim data ke Gemini AI untuk analisis teknikal mendalam
3. **Multiple Indicator Analysis**: Analisis 15+ technical indicators
4. **Risk Assessment**: Multiple probability assessments
5. **Exit Strategy Generation**: Detailed exit recommendations
6. **Technical Summary**: Generate comprehensive technical summary
7. **Enhanced Response**: Return hasil analisis dalam format JSON yang komprehensif
8. **Telegram Integration**: Send enhanced analysis to Telegram dengan format yang detail

## 🏗️ Project Structure

```
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes/
│   ├── config/
│   ├── models/
│   ├── services/
│   │   ├── yahoo_finance/
│   │   ├── gemini_ai/
│   │   └── trading_analysis/
│   └── utils/
├── pkg/
├── .env
├── env.example
├── go.mod
├── go.sum
├── README.md
└── requirements.md
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test ./internal/services/trading_analysis
```

## 🚀 Deployment

### Docker
```bash
# Build image
docker build -t swing-trading-signal .

# Run container
docker run -p 8080:8080 --env-file .env swing-trading-signal
```

### Production
```bash
# Build binary
go build -o bin/server cmd/server/main.go

# Run binary
./bin/server
```

## 🔒 Security

- Input validation untuk semua parameter
- Rate limiting untuk external APIs
- Error handling yang aman
- CORS configuration

## 📈 Monitoring

- Health check endpoint
- Structured logging dengan Logrus
- Error tracking dan monitoring
- Performance metrics

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

Jika ada pertanyaan atau masalah, silakan buat issue di repository ini.

## 🔮 Future Enhancements

- Backtesting engine
- Real-time alerts
- Web dashboard
- Mobile app
- Portfolio management
- Advanced technical indicators


WEBHOOK_RESPONSE=$(curl -s -X POST http://localhost:8080/telegram/set-webhook \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$WEBHOOK_URL\"}")


curl -X POST http://localhost:8080/api/v1/monitor-position \
  -H "Content-Type: application/json" \
  -d '{"url":""
  }'
