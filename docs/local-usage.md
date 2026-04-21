# AVmatrix Local Usage Guide

> Canonical rename spec cho local fork nay da duoc khoa tai [docs/avmatrix-canonical-spec.md](./avmatrix-canonical-spec.md).
>
> Canonical brand/command moi la `AVmatrix` / `avmatrix`.
> Trong repo dev nay, thu muc package van la `gitnexus/`, nen cac lenh local theo source tree van chay ben trong thu muc do.

Tai lieu nay dung de nhac nhanh cach chay AVmatrix theo kieu local-only tren may cua ban.

## 1. Muc tieu

AVmatrix trong repo nay duoc dung theo 3 phan:

- `avmatrix`: local bridge + CLI (package hien tai van nam trong `gitnexus/`)
- `gitnexus-web`: giao dien web local
- trinh duyet: hien thi graph, search, impact, chat, file navigation

Luong dung dung:

- terminal 1: chay local bridge
- terminal 2: chay `gitnexus-web`
- browser: mo giao dien de dung
- terminal 3 tuy chon: goi CLI truc tiep

`serve` o day la process local tren may ban, khong phai server cua ben thu ba.

## 2. Cai dat

Chay 1 lan de cai dependency:

```powershell
cd <repo-root>\gitnexus-shared
npm install

cd <repo-root>\gitnexus
npm install

cd <repo-root>\gitnexus-web
npm install
```

Luu y:

- `npm install` moi la buoc co the tai dependency tu npm/git sources
- `npm run serve` sau do chi chay code local trong repo nay

## 3. Chay local bridge

### Cach an toan, de hieu nhat

```powershell
cd <repo-root>\gitnexus
npm run serve
```

Hoac:

```powershell
cd <repo-root>\gitnexus
node dist/cli/index.js serve
```

Neu muon chay tu repo root:

```powershell
cd <repo-root>
npm run --prefix .\gitnexus serve
```

Khi thanh cong, ban se thay:

```text
AVmatrix server running on http://localhost:4747
```

## 4. Chay web UI

Mo terminal khac:

```powershell
cd <repo-root>\gitnexus-web
npm run dev
```

Mo browser:

```text
http://localhost:5173
```

## 5. Dung web UI de index repo local

Neu giao dien chua co graph:

1. nhap local path cua repo
2. bam analyze/index
3. doi qua trinh phan tich xong

Ban co the dung chinh repo nay:

```text
<repo-root>
```

Sau khi index xong, web UI dung de:

- xem graph codebase
- tim symbol
- xem impact
- theo doi quan he file/function
- chat voi local session

## 6. Dung CLI truc tiep

Mo terminal khac:

```powershell
cd <repo-root>\gitnexus
```

### Analyze repo

```powershell
node dist/cli/index.js analyze <repo-root>
```

Neu thu muc khong co `.git`, dung them `--skip-git`:

```powershell
node dist/cli/index.js analyze <project-root> --skip-git
```

Neu muon dat ten de sau nay de goi hon:

```powershell
node dist/cli/index.js analyze <project-root> --skip-git --name <repo-alias>
```

Luu y:

- `analyze` nhan `path` truc tiep, khong dung `--repo`
- `--repo` dung cho cac lenh doc/query sau khi repo da duoc index va dang ky

### Query

```powershell
node dist/cli/index.js query "local runtime" --repo <repo-alias>
```

### Context

```powershell
node dist/cli/index.js context "createServer" --repo <repo-alias>
```

### Impact

```powershell
node dist/cli/index.js impact "Function:gitnexus/src/server/api.ts:createServer" --repo <repo-alias>
```

### Detect changes

```powershell
node dist/cli/index.js detect-changes --repo <repo-alias> --scope unstaged
```

## 7. Neu dang o sai thu muc

Neu ban dang o `C:\Windows\System32` ma chay:

```powershell
npm run --prefix gitnexus serve
```

thi npm se tim:

```text
C:\Windows\System32\gitnexus\package.json
```

va bi loi.

Cach dung:

```powershell
cd <repo-root>
npm run --prefix .\gitnexus serve
```

hoac:

```powershell
cd <repo-root>\gitnexus
npm run serve
```

## 8. Phan biet cac lenh

### `npm run --prefix .\gitnexus serve`

- chay script local trong repo nay
- khong tu keo `gitnexus` tu GitHub
- khong tu `git clone`

### `avmatrix serve`

- sau khi CLI local da nam tren `PATH`, day la lenh canonical moi
- trong repo dev hien tai, van co the thay bang `cd gitnexus && npm run serve`

### `node dist/cli/index.js ...`

- chay ban build local trong repo nay
- khong tai package `gitnexus` tu npm
- on dinh hon `tsx` khi may ban gap loi `.ts` loader

Neu muon chac chan dang chay code local trong repo hien tai, uu tien:

```powershell
cd <repo-root>\gitnexus
npm run serve
```

## 9. Neu can nho nhanh

### Luong co ban

```powershell
# Terminal 1
cd <repo-root>\gitnexus
npm run serve

# Terminal 2
cd <repo-root>\gitnexus-web
npm run dev
```

Mo:

```text
http://localhost:5173
```

Nhap local repo path, analyze, roi dung graph/impact/chat/query.

## 10. Ghi nho

- `serve` la local bridge, khong phai server ben thu ba
- web UI chi la giao dien local de thao tac
- CLI dung khi can query/analyze/impact nhanh
- neu ban muon chac chan 100% dang chay code local, dung `cd <repo-root>\gitnexus && npm run serve`
