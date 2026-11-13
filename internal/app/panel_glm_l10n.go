package app

// GLM Coding Plan strings. English, Simplified Chinese, Traditional Chinese,
// Korean, Japanese, Russian, French — the same order as l10n.Languages. The
// tab label is a product name, so it is the same in every language.

func init() {
	extraTexts[xGLMTab] = [7]string{
		"GLM", "GLM", "GLM", "GLM", "GLM", "GLM", "GLM",
	}
	extraTexts[xGLMTitle] = [7]string{
		"Coding Plan quota", "Coding Plan 额度", "Coding Plan 額度", "Coding Plan 할당량",
		"Coding Plan のクォータ", "Квота Coding Plan", "Quota Coding Plan",
	}
	extraTexts[xGLMCredentialEnv] = [7]string{
		"Environment key", "环境变量密钥", "環境變數金鑰", "환경 변수 키",
		"環境変数のキー", "Ключ из переменной среды", "Clé d’environnement",
	}
	extraTexts[xGLMCredentialCodingPlan] = [7]string{
		"Coding Plan key", "编程套餐密钥", "編程方案金鑰", "코딩 플랜 키",
		"コーディングプランのキー", "Ключ Coding Plan", "Clé Coding Plan",
	}
	extraTexts[xGLMCredentialOAuth] = [7]string{
		"OAuth token", "OAuth 令牌", "OAuth 權杖", "OAuth 토큰",
		"OAuth トークン", "Токен OAuth", "Jeton OAuth",
	}
	extraTexts[xGLMCredentialNone] = [7]string{
		"No credential", "无凭据", "無憑據", "자격 증명 없음",
		"認証情報なし", "Нет учётных данных", "Aucun identifiant",
	}
	extraTexts[xGLMEmptyTitle] = [7]string{
		"No GLM credential", "没有 GLM 凭据", "沒有 GLM 憑據", "GLM 자격 증명이 없습니다",
		"GLM の認証情報がありません", "Нет учётных данных GLM", "Aucun identifiant GLM",
	}
	extraTexts[xGLMEmptyDetail] = [7]string{
		"Looked in %@. Sign in with `zcode login`, or set ZCODE_BIGMODEL_USAGE_API_KEY.",
		"已查找 %@。请用 `zcode login` 登录，或设置 ZCODE_BIGMODEL_USAGE_API_KEY。",
		"已查找 %@。請用 `zcode login` 登入，或設定 ZCODE_BIGMODEL_USAGE_API_KEY。",
		"%@ 위치를 확인했습니다. `zcode login`으로 로그인하거나 ZCODE_BIGMODEL_USAGE_API_KEY를 설정하세요.",
		"%@ を確認しました。`zcode login` でログインするか、ZCODE_BIGMODEL_USAGE_API_KEY を設定してください。",
		"Проверено: %@. Войдите через `zcode login` или задайте ZCODE_BIGMODEL_USAGE_API_KEY.",
		"Recherché dans %@. Connectez-vous avec `zcode login`, ou définissez ZCODE_BIGMODEL_USAGE_API_KEY.",
	}
	extraTexts[xGLMExpiredTitle] = [7]string{
		"The credential was rejected", "凭据被拒绝", "憑據被拒絕", "자격 증명이 거부되었습니다",
		"認証情報が拒否されました", "Учётные данные отклонены", "Identifiant refusé",
	}
	extraTexts[xGLMExpiredDetail] = [7]string{
		"The quota endpoint reports the credential as expired. Sign in again with `zcode login`; the store keeps no refresh token.",
		"额度接口返回凭据已过期。请重新 `zcode login`；凭据库中没有 refresh token。",
		"額度介面回報憑據已過期。請重新 `zcode login`；憑據庫中沒有 refresh token。",
		"할당량 엔드포인트가 자격 증명 만료를 보고합니다. `zcode login`으로 다시 로그인하세요. 저장소에는 refresh token이 없습니다.",
		"クォータのエンドポイントは認証情報の期限切れを返しています。`zcode login` で入り直してください。保存先に refresh token はありません。",
		"Эндпоинт квоты сообщает, что учётные данные истекли. Войдите снова через `zcode login`: refresh token в хранилище нет.",
		"L’endpoint de quota signale un identifiant expiré. Reconnectez-vous avec `zcode login` ; le stockage ne contient aucun refresh token.",
	}
	extraTexts[xGLMErrorTitle] = [7]string{
		"The quota could not be read", "无法读取额度", "無法讀取額度", "할당량을 읽을 수 없습니다",
		"クォータを読めません", "Квоту прочитать не удалось", "Quota illisible",
	}
	extraTexts[xGLMErrNetwork] = [7]string{
		"The endpoint did not answer.", "接口没有响应。", "介面沒有回應。", "엔드포인트가 응답하지 않았습니다.",
		"エンドポイントが応答しませんでした。", "Эндпоинт не ответил.", "L’endpoint n’a pas répondu.",
	}
	extraTexts[xGLMErrHTTP] = [7]string{
		"The endpoint answered with an HTTP error.", "接口返回了 HTTP 错误。", "介面回傳了 HTTP 錯誤。",
		"엔드포인트가 HTTP 오류를 반환했습니다.", "エンドポイントが HTTP エラーを返しました。",
		"Эндпоинт вернул ошибку HTTP.", "L’endpoint a renvoyé une erreur HTTP.",
	}
	extraTexts[xGLMErrCode] = [7]string{
		"The endpoint returned a business error.", "接口返回了业务错误。", "介面回傳了業務錯誤。",
		"엔드포인트가 비즈니스 오류를 반환했습니다.", "エンドポイントが業務エラーを返しました。",
		"Эндпоинт вернул бизнес-ошибку.", "L’endpoint a renvoyé une erreur métier.",
	}
	extraTexts[xGLMErrDecode] = [7]string{
		"The response was not the documented envelope.", "响应不是文档描述的信封结构。",
		"回應不是文件描述的信封結構。", "응답이 문서화된 봉투 형식이 아닙니다.",
		"応答はドキュメントのエンベロープ形式ではありません。",
		"Ответ не соответствует описанной обёртке.", "La réponse n’est pas l’enveloppe documentée.",
	}
	extraTexts[xGLMErrStore] = [7]string{
		"The credential store could not be read.", "无法读取凭据库。", "無法讀取憑據庫。",
		"자격 증명 저장소를 읽을 수 없습니다.", "認証情報の保存先を読めません。",
		"Хранилище учётных данных не прочитано.", "Le stockage des identifiants est illisible.",
	}
	extraTexts[xGLMErrDatabase] = [7]string{
		"The ZCode usage database could not be read.", "无法读取 ZCode 用量数据库。",
		"無法讀取 ZCode 用量資料庫。", "ZCode 사용량 데이터베이스를 읽을 수 없습니다.",
		"ZCode の使用量データベースを読めません。", "База использования ZCode не прочитана.",
		"La base d’usage ZCode est illisible.",
	}
	extraTexts[xGLMCostTitle] = [7]string{
		"API-equivalent cost", "API 等价成本", "API 等價成本", "API 환산 비용",
		"API 相当コスト", "Эквивалентная стоимость API", "Coût équivalent API",
	}
	extraTexts[xGLMCostEmpty] = [7]string{
		"The ZCode usage database has no model usage yet.",
		"ZCode 用量数据库还没有模型用量。",
		"ZCode 用量資料庫還沒有模型用量。",
		"ZCode 사용량 데이터베이스에 아직 모델 사용량이 없습니다.",
		"ZCode の使用量データベースにはまだモデルの使用量がありません。",
		"В базе использования ZCode пока нет данных по моделям.",
		"La base d’usage ZCode ne contient pas encore d’usage par modèle.",
	}
	extraTexts[xGLMCostTotal] = [7]string{
		"Total %@", "合计 %@", "合計 %@", "합계 %@", "合計 %@", "Всего %@", "Total %@",
	}
	extraTexts[xGLMCostTokensLine] = [7]string{
		"In %@ · Out %@ · Cache read %@ · %@ requests",
		"输入 %@ · 输出 %@ · 缓存读 %@ · %@ 次请求",
		"輸入 %@ · 輸出 %@ · 快取讀 %@ · %@ 次請求",
		"입력 %@ · 출력 %@ · 캐시 읽기 %@ · %@회 요청",
		"入力 %@ · 出力 %@ · キャッシュ読取り %@ · %@ リクエスト",
		"Вход %@ · Выход %@ · Чтение кэша %@ · запросов %@",
		"Entrée %@ · Sortie %@ · Lecture cache %@ · %@ requêtes",
	}
	extraTexts[xGLMCostUnpriced] = [7]string{
		"%@ models have no price and are left out of the total.",
		"%@ 个模型没有价格，未计入合计。",
		"%@ 個模型沒有價格，未計入合計。",
		"%@개 모델은 가격이 없어 합계에서 제외했습니다.",
		"%@ 件のモデルは価格がなく合計に含まれていません。",
		"Моделей без цены: %@; они не входят в сумму.",
		"%@ modèles sans prix, exclus du total.",
	}
	extraTexts[xGLMCostSourceNote] = [7]string{
		"Prices are USD per million tokens; a model without a price is not free.",
		"价格为每百万 token 的美元；没有价格的模型不等于免费。",
		"價格為每百萬 token 的美元；沒有價格的模型不等於免費。",
		"가격은 100만 토큰당 미국 달러입니다. 가격이 없는 모델이 무료인 것은 아닙니다.",
		"価格は 100 万トークンあたりの米ドルです。価格のないモデルが無料という意味ではありません。",
		"Цены в долларах США за миллион токенов; отсутствие цены не означает бесплатность.",
		"Les prix sont en dollars US par million de tokens ; l’absence de prix ne signifie pas la gratuité.",
	}
	extraTexts[xGLMFetched] = [7]string{
		"Fetched %@", "读取于 %@", "讀取於 %@", "가져온 시각 %@", "取得 %@", "Получено %@", "Récupéré %@",
	}
	extraTexts[xGLMDerived] = [7]string{
		"The share is derived from the remaining units.",
		"该占比由剩余数量推算。",
		"該佔比由剩餘數量推算。",
		"이 비율은 남은 수량에서 계산했습니다.",
		"この割合は残量から算出しています。",
		"Доля рассчитана по остатку.",
		"La part est déduite des unités restantes.",
	}
	extraTexts[xGLMQuotaSource] = [7]string{
		"Quota endpoint", "额度接口", "額度介面", "할당량 엔드포인트",
		"クォータのエンドポイント", "Эндпоинт квоты", "Endpoint de quota",
	}
	extraTexts[xGLMSourceNote] = [7]string{
		"Live vendor endpoint and the local ZCode usage database. The quota fields follow the ZCode source; a success response is not verified on this machine.",
		"实时厂商接口加本地 ZCode 用量库。额度字段依据 ZCode 源码；本机没有验证过成功响应。",
		"即時廠商介面加本機 ZCode 用量庫。額度欄位依據 ZCode 原始碼；本機沒有驗證過成功回應。",
		"실시간 공급사 엔드포인트와 로컬 ZCode 사용량 데이터베이스입니다. 할당량 필드는 ZCode 소스를 따르며, 이 컴퓨터에서 성공 응답은 검증되지 않았습니다.",
		"ベンダーのライブエンドポイントとローカル ZCode 使用量データベースです。クォータの項目は ZCode のソースに従っており、このマシンでは成功応答を確認していません。",
		"Живой эндпоинт вендора и локальная база использования ZCode. Поля квоты взяты из исходников ZCode; успешный ответ на этой машине не проверен.",
		"Endpoint vendeur en direct et base d’usage ZCode locale. Les champs de quota suivent les sources de ZCode ; aucune réponse de succès n’a été vérifiée sur cette machine.",
	}
}

const (
	xGLMTab                  textKey = "glmTab"
	xGLMTitle                textKey = "glmTitle"
	xGLMCredentialEnv        textKey = "glmCredentialEnv"
	xGLMCredentialCodingPlan textKey = "glmCredentialCodingPlan"
	xGLMCredentialOAuth      textKey = "glmCredentialOAuth"
	xGLMCredentialNone       textKey = "glmCredentialNone"
	xGLMEmptyTitle           textKey = "glmEmptyTitle"
	xGLMEmptyDetail          textKey = "glmEmptyDetail"
	xGLMExpiredTitle         textKey = "glmExpiredTitle"
	xGLMExpiredDetail        textKey = "glmExpiredDetail"
	xGLMErrorTitle           textKey = "glmErrorTitle"
	xGLMErrNetwork           textKey = "glmErrNetwork"
	xGLMErrHTTP              textKey = "glmErrHTTP"
	xGLMErrCode              textKey = "glmErrCode"
	xGLMErrDecode            textKey = "glmErrDecode"
	xGLMErrStore             textKey = "glmErrStore"
	xGLMErrDatabase          textKey = "glmErrDatabase"
	xGLMCostTitle            textKey = "glmCostTitle"
	xGLMCostEmpty            textKey = "glmCostEmpty"
	xGLMCostTotal            textKey = "glmCostTotal"
	xGLMCostTokensLine       textKey = "glmCostTokensLine"
	xGLMCostUnpriced         textKey = "glmCostUnpriced"
	xGLMCostSourceNote       textKey = "glmCostSourceNote"
	xGLMFetched              textKey = "glmFetched"
	xGLMDerived              textKey = "glmDerived"
	xGLMQuotaSource          textKey = "glmQuotaSource"
	xGLMSourceNote           textKey = "glmSourceNote"
)
