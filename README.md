# tfe-cli

HCP Terraform の state を管理する CLI ツール。

## 環境変数

| 変数名 | 説明 |
|---|---|
| `TFC_API_TOKEN` | HCP Terraform の API トークン |
| `TFC_ORGANIZATION` | 組織名 |
| `TFC_WORKSPACE_NAME` | ワークスペース名 |

## コマンド

```
tfe status
```
認証状態とアカウント情報を表示する。

```
tfe state list
```
直近 10 件の state バージョン一覧を表示する。

```
tfe state show [<latest|sv-...>]
```
state のメタデータを表示する。バージョン省略時は最新を表示。

```
tfe state download [<latest|sv-...>] [-o <output_file>]
```
state をダウンロードする。バージョン省略時は最新。`-o` 省略時は `terraform.tfstate` または `sv-....tfstate` に保存される。

```
tfe state upload [file_path]
```
state をアップロードする。ファイルパス省略時は `terraform.tfstate` を使用。

```
tfe actions lock
tfe actions unlock
```
ワークスペースを手動で lock / unlock する。
