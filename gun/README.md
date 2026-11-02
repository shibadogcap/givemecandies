# gun

負荷試験用のクライアント。`/c` に POST を投げ続けて、req/s とレイテンシを出す。

```sh
go build -o gun .
./gun -url https://givemecandies.shibadogcap.com -c 128 -d 15 -amount 1
./gun -url https://givemecandies.shibadogcap.com -c 160 -d 30 -amount 2000000000000 -name 本気パンチ
```

| フラグ | 既定 | 意味 |
| --- | --- | --- |
| `-url` | 必須 | ベース URL |
| `-c` | 64 | 並列数 |
| `-d` | 20 | 秒 |
| `-amount` | 1 | 1 リクエストのお菓子の数 |
| `-name` | 空 | 送信者名（省略するとランキングに載らない） |

1 プロセスで 4,000 req/s 前後まで出る。それ以上はサーバー側かネットワークで頭打ちになる。
同じマシンで複数立てても合計は伸びないので、上限を測るときは別ホストからも撃つ。
