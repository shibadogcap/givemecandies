#!/usr/bin/env python3
"""docs/data.json から docs/index.html を作る。数字は丸めない。"""
import json, datetime, pathlib

root = pathlib.Path(__file__).resolve().parent.parent
d = json.loads((root / "docs" / "data.json").read_text())
data_js = json.dumps(d, ensure_ascii=False, separators=(",", ":"))

HTML = r"""<!doctype html>
<html lang="ja"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>givemecandies の記録</title>
<meta name="description" content="2026年ハロウィンに公開した「お菓子をあげる」ページの記録。65,666,798 リクエスト。">
<meta property="og:type" content="website">
<meta property="og:title" content="givemecandies の記録">
<meta property="og:description" content="65,666,798 リクエスト、133,990,739,323,320,559,165 個。ハロウィンの 31 時間。">
<link rel="icon" href="favicon.svg" type="image/svg+xml">
<meta name="twitter:card" content="summary">
<style>
:root{--cream:#FFF9EC;--card:#fff;--yellow:#FFDD00;--ink:#2F2A24;--brown:#6B5B4B;--line:#EFE5D2;--soft:#FFFDF6}
*{box-sizing:border-box}
body{margin:0;background:var(--cream);color:var(--ink);
 font-family:-apple-system,BlinkMacSystemFont,"Hiragino Kaku Gothic ProN","Noto Sans JP",sans-serif;
 background-image:radial-gradient(#F1E7D2 1px,transparent 1px);background-size:18px 18px;
 padding:28px 16px 48px;display:flex;justify-content:center}
.wrap{width:100%;max-width:760px}
.brand{display:flex;align-items:center;gap:8px;justify-content:center;margin-bottom:18px;font-weight:800;color:var(--brown);font-size:15px}
.brand .dot{width:18px;height:18px;border-radius:50%;background:var(--yellow);display:grid;place-items:center;font-size:10px}
.card{background:var(--card);border:1px solid var(--line);border-radius:22px;padding:28px 24px;box-shadow:0 18px 40px -24px #6B5B4B55,0 2px 0 #EFE5D2;margin-bottom:14px}
h1{margin:0 0 6px;font-size:20px;font-weight:850;letter-spacing:-.01em;text-align:center}
.sub{margin:0 0 22px;text-align:center;font-size:12.5px;color:var(--brown);line-height:1.7}
.hero{text-align:center;padding:6px 0 4px}
.num{font-variant-numeric:tabular-nums;font-size:clamp(19px,4.4vw,33px);font-weight:850;letter-spacing:-.01em;line-height:1.25;
 word-break:break-all;margin:0}
.num .u{display:block;font-size:12.5px;font-weight:650;color:var(--brown);letter-spacing:0;margin-top:6px}
.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:22px}
.cell{background:var(--soft);border:1px solid var(--line);border-radius:14px;padding:12px 10px;text-align:center}
.cell .k{font-size:10.5px;color:var(--brown);letter-spacing:.04em}
.cell .v{font-size:15px;font-weight:750;font-variant-numeric:tabular-nums;margin-top:3px;word-break:break-all}
h2{font-size:12px;font-weight:800;color:var(--brown);letter-spacing:.06em;margin:0 0 12px;text-transform:uppercase}
.chart svg{display:block;width:100%;height:150px}
.axis{display:flex;justify-content:space-between;font-size:10.5px;color:#a2947d;margin-top:6px;font-variant-numeric:tabular-nums}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;font-size:10.5px;color:var(--brown);font-weight:700;padding:0 0 8px;border-bottom:1px solid var(--line)}
th.r,td.r{text-align:right;font-variant-numeric:tabular-nums}
td{padding:8px 0;border-bottom:1px solid #F7F0E1}
tr:last-child td{border-bottom:0}
.rank{color:#b3a48c;width:26px;font-variant-numeric:tabular-nums}
.nm{font-weight:600;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:0}
td.r{font-size:12px;padding-left:10px;white-space:nowrap}
.cols{display:grid;grid-template-columns:1fr 1fr;gap:22px}
ul{margin:0;padding:0;list-style:none}
ul li{display:flex;justify-content:space-between;gap:10px;padding:6px 0;border-bottom:1px solid #F7F0E1;font-size:12.5px}
ul li:last-child{border-bottom:0}
ul .n{color:var(--brown);font-variant-numeric:tabular-nums}
.note{font-size:11.5px;color:var(--brown);line-height:1.9}
.note b{color:var(--ink)}
@media (max-width:560px){.cols{grid-template-columns:1fr}.grid{grid-template-columns:1fr}}
</style></head>
<body><div class="wrap">
<div class="brand"><span class="dot">🍬</span>givemecandies</div>

<div class="card">
  <div class="hero">
    <h1>ハロウィンの記録</h1>
    <p class="sub" id="period"></p>
    <p class="num"><span id="total">0</span><span class="u" id="unit"></span></p>
    <div class="grid">
      <div class="cell"><div class="k">リクエスト</div><div class="v" id="reqs">0</div></div>
      <div class="cell"><div class="k">ピーク</div><div class="v" id="peak">0</div></div>
      <div class="cell"><div class="k">書き込み</div><div class="v" id="flushes">0</div></div>
    </div>
  </div>
</div>

<div class="card">
  <h2>10 分あたりのお菓子（縦軸は対数）</h2>
  <div class="chart"><svg id="spark" viewBox="0 0 700 150" preserveAspectRatio="none"></svg></div>
  <div class="axis"><span id="t0"></span><span id="t1"></span></div>
</div>

<div class="card">
  <h2>お菓子をくれた人</h2>
  <table><thead><tr><th>#</th><th>名前</th><th class="r">お菓子</th></tr></thead><tbody id="rows"></tbody></table>
</div>

<div class="card">
  <h2>どこから来たか</h2>
  <div class="cols">
    <div><h2 style="margin-bottom:8px">国</h2><ul id="ctry"></ul></div>
    <div><h2 style="margin-bottom:8px">クライアント</h2><ul id="uas"></ul></div>
  </div>
  <div class="cols" style="margin-top:18px">
    <div><h2 style="margin-bottom:8px">パス</h2><ul id="paths"></ul></div>
    <div><h2 style="margin-bottom:8px">ステータス</h2><ul id="st"></ul></div>
  </div>
</div>

<div class="card">
  <h2>これは何だったか</h2>
  <p class="note">
    2026 年のハロウィンに、OCI の Always Free（Ampere A1 2 OCPU / 12GB）で動かした「任意個のお菓子を送れる」ページの記録。
    1 リクエストで最大 2 兆個を送れる設計なので、65,666,798 リクエストが <b>133,990,739,323,320,559,165 個</b> になった。
    書き込みはメモリで加算して 1 秒に 1 回だけディスクへ、応答は本文なしの 204。カウンタは 64bit に収まらないので可変長整数で持っている。
    31 時間で落ちず、再起動も 0 回。エラーは 775 件（0.001%）で、すべてクライアント由来の 400。
  </p>
  <p class="note" style="margin-top:10px">
    数字はすべて一の位まで、丸めずに載せている。上の大きい数字の下に出る「京」表記だけは読みやすいように併記したもの。
    アクセス元の IP アドレスは集計に使っただけで、このページには載せていない。名前は送信者が入力したもの。
  </p>
</div>

<div class="card" style="padding:16px 24px">
  <p class="note" style="margin:0">静的なアーカイブ。データは <span id="gen"></span> 時点。</p>
</div>
</div>
<script>
const D = __DATA__;
const $=s=>document.querySelector(s);
const UNITS=[[10n**28n,"穣"],[10n**24n,"秭"],[10n**20n,"垓"],[10n**16n,"京"],[10n**12n,"兆"],[10n**8n,"億"],[10n**4n,"万"]];
function approx(v){ for(const [s,u] of UNITS) if(v>=s){ const w=v/s,f=(v%s)*1000n/s;
    return (w.toLocaleString('ja-JP')+(f?'.'+String(f).padStart(3,'0').replace(/0+$/,''):''))+' '+u; } return ''; }
function exact(v){ return v.toLocaleString('ja-JP'); }
function ts(t){ return new Date(t*1000).toLocaleString('ja-JP',{timeZone:'Asia/Tokyo',month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit'}); }

const total = BigInt(D.meta.total);
$('#total').textContent = exact(total);
$('#unit').textContent = '（' + approx(total) + '）';
$('#reqs').textContent = exact(BigInt(D.meta.requests));
$('#peak').textContent = exact(BigInt(D.meta.peak)) + ' req/s';
$('#flushes').textContent = exact(BigInt(D.meta.flushes));
$('#period').textContent = ts(D.meta.first)+' から '+ts(D.meta.last)+' まで（'+Math.round(D.meta.seconds/3600)+' 時間）';
$('#t0').textContent = ts(D.meta.first); $('#t1').textContent = ts(D.meta.last);
$('#gen').textContent = ts(D.meta.generated);

(function(){
  // log scale: the quiet hours are ~1e15 and the bursts ~1e18, so linear
  // collapses everything but the peak.
  const w=700,h=150,pad=4;
  const vals=D.timeline.map(x=>x.c||1);
  const logs=vals.map(v=>Math.log10(v));
  const lo=Math.floor(Math.min(...logs)), hi=Math.ceil(Math.max(...logs));
  const span=Math.max(hi-lo,1);
  const yOf=v=>h-pad-(h-2*pad)*(Math.log10(Math.max(v,Math.pow(10,lo)))-lo)/span;
  const xOf=i=>pad+(w-2*pad)*i/Math.max(vals.length-1,1);
  const lab=e=>{ if(e>=16) return (10**(e-16)).toLocaleString('ja-JP')+'京';
                 return (10**(e-12)).toLocaleString('ja-JP')+'兆'; };
  let grid='';
  for(let e=lo;e<=hi;e++){
    const y=yOf(Math.pow(10,e)).toFixed(1);
    grid+='<line x1="'+pad+'" y1="'+y+'" x2="'+(w-pad)+'" y2="'+y+'" stroke="#EFE5D2" stroke-width="1" stroke-dasharray="'+(e%2?'2 4':'0')+'"/>';
    if(e%2===0||e===hi) grid+='<text x="'+(pad+2)+'" y="'+(y-3)+'" font-size="9" fill="#b3a48c" stroke="#fff" stroke-width="2.5" paint-order="stroke" stroke-linejoin="round">'+lab(e)+'</text>';
  }
  let path='';
  vals.forEach((v,i)=>{ const x=xOf(i).toFixed(1), y=yOf(v).toFixed(1); path+=(i?'L':'M')+x+' '+y; });
  const area=path+'L'+(w-pad)+' '+(h-pad)+'L'+pad+' '+(h-pad)+'Z';
  $('#spark').innerHTML='<defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#FFDD00" stop-opacity=".8"/><stop offset="1" stop-color="#FFDD00" stop-opacity=".06"/></linearGradient></defs>'+
    '<path d="'+area+'" fill="url(#g)"/>'+grid+'<path d="'+path+'" fill="none" stroke="#E8C800" stroke-width="1.6"/>';
})();

function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
$('#rows').innerHTML = D.senders.slice(0,15).map((s,i)=>
  '<tr><td class="rank">'+(i+1)+'</td><td class="nm">'+esc(s.n)+'</td><td class="r">'+exact(BigInt(s.t))+'</td></tr>').join('');

function ul(el,arr){ el.innerHTML=arr.map(a=>'<li><span>'+esc(a.k)+'</span><span class="n">'+exact(BigInt(a.n))+'</span></li>').join(''); }
ul($('#ctry'), Object.entries(D.countries).sort((a,b)=>b[1]-a[1]).slice(0,8).map(([k,n])=>({k,n})));
ul($('#uas'), D.uas.map(x=>({k:x.k.slice(0,34),n:x.n})));
ul($('#paths'), D.paths.map(x=>({k:x.k,n:x.n})));
ul($('#st'), Object.entries(D.status).sort((a,b)=>b[1]-a[1]).map(([k,n])=>({k,n})));
</script>
</body></html>"""

(root / "docs" / "index.html").write_text(HTML.replace("__DATA__", data_js), encoding="utf-8")
print("wrote docs/index.html", (root / "docs" / "index.html").stat().st_size, "bytes")
