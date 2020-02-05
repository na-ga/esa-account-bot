# esa-account-bot

esa.io のアカウントを管理する Slack Bot です

## Setting

次の環境変数を指定します

- **BOT_ID**: BOT の Id を指定する
- **BOT_TOKEN**: BOT の Token を指定する
- **CHANNEL_ID**: BOT を動かす Channel Id を指定する
- **VERIFICATION_TOKEN**: Application の Verification Token を指定する
- **ESA_TOKEN**: ESA Owner アカウントの Token を指定する
- **ESA_TEAM_NAME**: ESA のチーム名を指定する
- **ADMIN_IDS**: 管理者の Slack User ID をカンマ区切りで指定する

必要であれば、次の環境変数を指定します

- **ADMIN_GROUP_ID**: 管理者の Slack Group ID を指定する
- **ALLOW_EMAIL_DOMAINS**: 許可するメールアドレスのドメインをカンマ区切りで指定する
- **ORGANIZATIONS**: 想定される利用者の所属組織をカンマ区切りで指定する

## Feature

次のオペレーションを Slack Bot で実現します

- 管理者の承認後を得て、指定したメールアドレスに招待メールを送信する
- 管理者の承認後を得て、指定したアカウントをチームから削除する
- 管理者の承認後を得て、指定した期間においてログインしていないアカウントをチームから削除する

![usage](/usage.png)


