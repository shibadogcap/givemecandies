# givemecandies

2026 年のハロウィンに公開した「任意個のお菓子を送れる」ページのソースと記録。

- 記録: 65,666,798 リクエスト / 133 京 / 31 時間 / 再起動 0
- ピーク: 4,300 req/s（OCI Always Free Ampere A1、2 OCPU / 12GB）
- アーカイブ: `docs/`（GitHub Pages で公開）

## 設計

| 判断 | 理由 |
| --- | --- |
| 応答は 204、本文なし | 1 リクエストあたりの転送を最小にする。表示はクライアントが楽観更新 |
| 加算はメモリ、ディスクへは 1 秒に 1 回 | 1 票 1 行で書くと SSD と課金が死ぬ。65,666,798 リクエストに対して書き込みは 65,757 行 |
| カウンタは `big.Int` | 1 リクエスト最大 2 兆。64bit は 1.8e19 で、最大速だと 40 分で折れる。実際に 1.33e20 まで到達した |
| 名前は正規化して比較 | 全角→半角、制御/書式/結合文字を除去、24 文字上限。見えない文字で複製を作れなくする |
| 新規名は IP あたり 1 分 5 個 | 連番で名前を大量に作る攻撃への耐性。1000 個未満は 10 分で失効、表は 5 万件で頭打ち |
| オリジンは公開ポート無し | cloudflared の外向きトンネルだけ。 volumetric は Cloudflare のエッジで受ける |
| レート制限は既定で無効 | 正面から受けて容量で測るため。`CANDY_LIMIT_RATE` で有効化できる |

## 動かす

```sh
go build -o candy .
CANDY_ADDR=:8080 CANDY_DATA=/var/lib/candy ./candy
```

| 環境変数 | 既定 | 意味 |
| --- | --- | --- |
| `CANDY_ADDR` | `:8080` | 待受アドレス（SO_REUSEPORT） |
| `CANDY_DATA` | `/var/lib/candy` | ログと送信者表の置き場所 |
| `CANDY_MAX_AMOUNT` | `2000000000000` | 1 リクエストの上限 |
| `CANDY_LIMIT_RATE` | `0` | 1 秒あたりの票数（0 で無効） |
| `CANDY_LIMIT_BURST` | `2000` | 同上のバースト |
| `CANDY_IDEMPOTENCY` | `1` | リクエスト ID で二重送信を弾く |

## エンドポイント

| パス | 用途 |
| --- | --- |
| `GET /` | ページ |
| `POST /c` | `{"name":"...","amount":N,"rid":"..."}` → 204 |
| `GET /api/stats` | 累計、rps、ピーク、IP 数 |
| `GET /api/ranking` | 送信者上位 100 |
| `GET /api/attack` | ハニーポット側の集計（UA、パス、国） |
| `GET /attack` | その表示ページ |

## デプロイ

`deploy/deploy.sh` は新しいバイナリを先に同じポートへ bind（SO_REUSEPORT）させ、
`systemd` を restart してから待機プロセスを落とす。切り替え中のリクエストは待機側が受ける。
`deploy/snapshot.sh` は本番データの退避と復元。

## ライセンス

MIT
