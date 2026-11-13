package app

// Gemini Code Assist strings. English, Simplified Chinese, Traditional
// Chinese, Korean, Japanese, Russian, French — the same order as
// l10n.Languages. The tab label is a product name, so it is the same in every
// language.

func init() {
	extraTexts[xGeminiTab] = [7]string{
		"Gemini", "Gemini", "Gemini", "Gemini", "Gemini", "Gemini", "Gemini",
	}
	extraTexts[xGeminiTitle] = [7]string{
		"Code Assist quota", "Code Assist 额度", "Code Assist 額度", "Code Assist 할당량", "Code Assist クォータ", "Квота Code Assist", "Quota Code Assist",
	}
	extraTexts[xGeminiNoCredsTitle] = [7]string{
		"Not signed in", "未登录", "未登入", "로그인되지 않음", "未ログイン", "Нет входа", "Non connecté",
	}
	extraTexts[xGeminiNoCredsDetail] = [7]string{
		"Run the Gemini CLI and sign in with Google. This page reads %@ and never stores a token.",
		"请运行 Gemini CLI 并用 Google 账号登录。本页只读取 %@，不会保存 token。",
		"請執行 Gemini CLI 並用 Google 帳號登入。本頁只讀取 %@，不會儲存 token。",
		"Gemini CLI를 실행해 Google 계정으로 로그인하세요. 이 페이지는 %@만 읽으며 토큰을 저장하지 않습니다.",
		"Gemini CLI を実行して Google でログインしてください。このページは %@ を読むだけで、トークンは保存しません。",
		"Запустите Gemini CLI и войдите через Google. Эта страница читает только %@ и не сохраняет токен.",
		"Lancez le Gemini CLI et connectez-vous avec Google. Cette page lit seulement %@ et ne conserve aucun jeton.",
	}
	extraTexts[xGeminiExpiredTitle] = [7]string{
		"Sign-in expired", "登录已过期", "登入已過期", "로그인이 만료되었습니다", "ログインの有効期限切れ", "Срок входа истёк", "Session expirée",
	}
	extraTexts[xGeminiExpiredDetail] = [7]string{
		"The refresh token no longer works. Sign in again with the Gemini CLI, and this page will reread %@.",
		"刷新令牌已失效。请用 Gemini CLI 重新登录，本页会重新读取 %@。",
		"重新整理權杖已失效。請用 Gemini CLI 重新登入，本頁會重新讀取 %@。",
		"갱신 토큰이 더 이상 작동하지 않습니다. Gemini CLI로 다시 로그인하면 이 페이지가 %@를 다시 읽습니다.",
		"リフレッシュトークンが無効になりました。Gemini CLI で再ログインすると、このページは %@ を読み直します。",
		"Токен обновления больше не работает. Войдите снова через Gemini CLI — страница перечитает %@.",
		"Le jeton de rafraîchissement ne fonctionne plus. Reconnectez-vous avec le Gemini CLI ; cette page relira %@.",
	}
	extraTexts[xGeminiNoSecretDetail] = [7]string{
		"This page does not renew a sign-in by itself: run the Gemini CLI once and it rewrites %@. To let this page renew it, set GEMINI_OAUTH_CLIENT_ID and GEMINI_OAUTH_CLIENT_SECRET.",
		"本页不会自行续期登录：运行一次 Gemini CLI，它会重写 %@。若要让本页自行续期，请设置 GEMINI_OAUTH_CLIENT_ID 与 GEMINI_OAUTH_CLIENT_SECRET。",
		"本頁不會自行續期登入：執行一次 Gemini CLI，它會重寫 %@。若要讓本頁自行續期，請設定 GEMINI_OAUTH_CLIENT_ID 與 GEMINI_OAUTH_CLIENT_SECRET。",
		"이 페이지는 로그인을 스스로 갱신하지 않습니다. Gemini CLI를 한 번 실행하면 %@를 다시 씁니다. 이 페이지가 갱신하게 하려면 GEMINI_OAUTH_CLIENT_ID와 GEMINI_OAUTH_CLIENT_SECRET을 설정하세요.",
		"このページはログインを自動更新しません。Gemini CLI を一度実行すると %@ が書き直されます。このページに更新させるには GEMINI_OAUTH_CLIENT_ID と GEMINI_OAUTH_CLIENT_SECRET を設定してください。",
		"Эта страница не обновляет вход сама: запустите Gemini CLI один раз, и он перезапишет %@. Чтобы страница обновляла его, задайте GEMINI_OAUTH_CLIENT_ID и GEMINI_OAUTH_CLIENT_SECRET.",
		"Cette page ne renouvelle pas la connexion d’elle-même : lancez une fois le Gemini CLI et il réécrira %@. Pour que cette page le fasse, définissez GEMINI_OAUTH_CLIENT_ID et GEMINI_OAUTH_CLIENT_SECRET.",
	}
	extraTexts[xGeminiNoSubscriptionTitle] = [7]string{
		"No Code Assist quota", "没有 Code Assist 额度", "沒有 Code Assist 額度", "Code Assist 할당량 없음", "Code Assist クォータなし", "Нет квоты Code Assist", "Aucun quota Code Assist",
	}
	extraTexts[xGeminiNoSubscriptionDetail] = [7]string{
		"The service answered, but this account or this client may not use the Code Assist quota endpoint. A paid Code Assist subscription, or a newer Google client, is required. This is not a reading error.",
		"服务已应答，但该账号或该客户端无权使用 Code Assist 额度接口。需要付费 Code Assist 订阅，或改用新版 Google 客户端。这不是读取错误。",
		"服務已回應，但此帳號或此用戶端無權使用 Code Assist 額度介面。需要付費 Code Assist 訂閱，或改用新版 Google 用戶端。這不是讀取錯誤。",
		"서비스가 응답했지만 이 계정이나 이 클라이언트는 Code Assist 할당량 엔드포인트를 쓸 수 없습니다. 유료 Code Assist 구독이나 최신 Google 클라이언트가 필요합니다. 읽기 오류가 아닙니다.",
		"サービスは応答しましたが、このアカウントまたはクライアントは Code Assist クォータのエンドポイントを利用できません。有料の Code Assist サブスクリプションか、新しい Google クライアントが必要です。読み取りエラーではありません。",
		"Сервис ответил, но эта учётная запись или этот клиент не может использовать эндпоинт квоты Code Assist. Нужна платная подписка Code Assist или более новый клиент Google. Это не ошибка чтения.",
		"Le service a répondu, mais ce compte ou ce client ne peut pas utiliser l’endpoint de quota Code Assist. Un abonnement Code Assist payant ou un client Google plus récent est requis. Ce n’est pas une erreur de lecture.",
	}
	extraTexts[xGeminiErrorTitle] = [7]string{
		"Reading failed", "读取失败", "讀取失敗", "읽기 실패", "読み取りに失敗しました", "Чтение не удалось", "Échec de la lecture",
	}
	extraTexts[xGeminiErrNetwork] = [7]string{
		"The endpoint could not be reached.", "无法连接接口。", "無法連線介面。", "엔드포인트에 연결할 수 없습니다.", "エンドポイントに接続できませんでした。", "Не удалось связаться с эндпоинтом.", "L’endpoint est injoignable.",
	}
	extraTexts[xGeminiErrHTTP] = [7]string{
		"The endpoint answered HTTP %@.", "接口返回 HTTP %@。", "介面回傳 HTTP %@。", "엔드포인트가 HTTP %@를 반환했습니다.", "エンドポイントが HTTP %@ を返しました。", "Эндпоинт ответил HTTP %@.", "L’endpoint a répondu HTTP %@.",
	}
	extraTexts[xGeminiErrResponse] = [7]string{
		"The endpoint answered with a body this page cannot read.", "接口返回了本页无法解析的内容。", "介面回傳了本頁無法解析的內容。", "엔드포인트가 이 페이지가 읽을 수 없는 본문을 반환했습니다.", "エンドポイントが解釈できない内容を返しました。", "Эндпоинт вернул тело, которое страница не может прочитать.", "L’endpoint a renvoyé un corps illisible pour cette page.",
	}
	extraTexts[xGeminiErrCreds] = [7]string{
		"The credential file could not be read.", "无法读取凭据文件。", "無法讀取憑證檔案。", "자격 증명 파일을 읽을 수 없습니다.", "資格情報ファイルを読めません。", "Файл учётных данных не прочитан.", "Le fichier d’identifiants est illisible.",
	}
	extraTexts[xGeminiErrSignIn] = [7]string{
		"The saved sign-in was refused. Sign in again with the Gemini CLI.", "已保存的登录被拒绝。请用 Gemini CLI 重新登录。", "已儲存的登入被拒絕。請用 Gemini CLI 重新登入。", "저장된 로그인이 거부되었습니다. Gemini CLI로 다시 로그인하세요.", "保存されたログインが拒否されました。Gemini CLI で再ログインしてください。", "Сохранённый вход отклонён. Войдите снова через Gemini CLI.", "La connexion enregistrée a été refusée. Reconnectez-vous avec le Gemini CLI.",
	}
	extraTexts[xGeminiErrGeneric] = [7]string{
		"The reading failed.", "读取失败。", "讀取失敗。", "읽기에 실패했습니다.", "読み取りに失敗しました。", "Чтение не удалось.", "La lecture a échoué.",
	}
	extraTexts[xGeminiReason] = [7]string{
		"Reason %@", "原因 %@", "原因 %@", "사유 %@", "理由 %@", "Причина %@", "Raison %@",
	}
	extraTexts[xGeminiNoBuckets] = [7]string{
		"The endpoint returned no quota bucket for any model.", "接口没有返回任何模型的额度桶。", "介面沒有回傳任何模型的額度桶。", "엔드포인트가 어떤 모델의 할당량 버킷도 반환하지 않았습니다.", "どのモデルのクォータバケットも返されませんでした。", "Эндпоинт не вернул ни одного бакета квоты.", "L’endpoint n’a renvoyé aucun seau de quota.",
	}
	extraTexts[xGeminiRequests] = [7]string{
		"Remaining %@ of %@ requests", "剩余 %@ / %@ 次请求", "剩餘 %@ / %@ 次請求", "남은 요청 %@ / %@", "残り %@ / %@ リクエスト", "Осталось %@ из %@ запросов", "%@ requêtes restantes sur %@",
	}
	extraTexts[xGeminiCostEmpty] = [7]string{
		"No Gemini CLI session has recorded token usage yet.", "尚未有任何 Gemini CLI 会话记录 token 用量。", "尚未有任何 Gemini CLI 工作階段記錄 token 用量。", "아직 토큰 사용량을 기록한 Gemini CLI 세션이 없습니다.", "トークン使用量を記録した Gemini CLI セッションはまだありません。", "Ни одна сессия Gemini CLI ещё не записала расход токенов.", "Aucune session Gemini CLI n’a encore enregistré de tokens.",
	}
	extraTexts[xGeminiCostEstimate] = [7]string{
		"Estimated from the model family; this exact model has no published price.",
		"按同系列模型估算：该型号没有公开价目。",
		"依同系列模型估算：此型號沒有公開價目。",
		"같은 계열 모델 기준 추정치입니다. 이 모델에는 공개된 가격이 없습니다.",
		"同系列モデルからの概算です。この型番には公開価格がありません。",
		"Оценка по семейству модели: у этой версии нет опубликованной цены.",
		"Estimé d’après la famille du modèle ; cette version n’a pas de prix publié.",
	}
	extraTexts[xGeminiSessions] = [7]string{
		"From %@ local session file(s).", "来自 %@ 个本机会话文件。", "來自 %@ 個本機工作階段檔案。", "로컬 세션 파일 %@개 기준입니다.", "ローカルセッションファイル %@ 件から集計しています。", "Из %@ локальных файлов сессий.", "D’après %@ fichiers de session locaux.",
	}
	extraTexts[xGeminiSourceNote] = [7]string{
		"Quota read live from the Gemini Code Assist endpoints; cost read from the local CLI chat logs.",
		"额度实时读取 Gemini Code Assist 接口；成本读取本机 CLI 聊天日志。",
		"額度即時讀取 Gemini Code Assist 介面；成本讀取本機 CLI 聊天日誌。",
		"할당량은 Gemini Code Assist 엔드포인트에서 실시간으로, 비용은 로컬 CLI 채팅 로그에서 읽습니다.",
		"クォータは Gemini Code Assist のエンドポイントから、コストはローカルの CLI チャットログから読み取ります。",
		"Квота читается с эндпоинтов Gemini Code Assist, стоимость — из локальных журналов чата CLI.",
		"Le quota est lu en direct sur les endpoints Gemini Code Assist ; le coût, dans les journaux de chat locaux du CLI.",
	}
}

const (
	xGeminiTab                  textKey = "geminiTab"
	xGeminiTitle                textKey = "geminiTitle"
	xGeminiNoCredsTitle         textKey = "geminiNoCredsTitle"
	xGeminiNoCredsDetail        textKey = "geminiNoCredsDetail"
	xGeminiExpiredTitle         textKey = "geminiExpiredTitle"
	xGeminiExpiredDetail        textKey = "geminiExpiredDetail"
	xGeminiNoSecretDetail       textKey = "geminiNoSecretDetail"
	xGeminiNoSubscriptionTitle  textKey = "geminiNoSubscriptionTitle"
	xGeminiNoSubscriptionDetail textKey = "geminiNoSubscriptionDetail"
	xGeminiErrorTitle           textKey = "geminiErrorTitle"
	xGeminiErrNetwork           textKey = "geminiErrNetwork"
	xGeminiErrHTTP              textKey = "geminiErrHTTP"
	xGeminiErrResponse          textKey = "geminiErrResponse"
	xGeminiErrCreds             textKey = "geminiErrCreds"
	xGeminiErrSignIn            textKey = "geminiErrSignIn"
	xGeminiErrGeneric           textKey = "geminiErrGeneric"
	xGeminiReason               textKey = "geminiReason"
	xGeminiNoBuckets            textKey = "geminiNoBuckets"
	xGeminiRequests             textKey = "geminiRequests"
	xGeminiCostEmpty            textKey = "geminiCostEmpty"
	xGeminiCostEstimate         textKey = "geminiCostEstimate"
	xGeminiSessions             textKey = "geminiSessions"
	xGeminiSourceNote           textKey = "geminiSourceNote"
)
