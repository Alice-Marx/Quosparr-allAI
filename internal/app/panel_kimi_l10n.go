package app

// Kimi Code quota strings. The tab label is a product name, so it is the same in
// every language. The order is l10n.Languages: English, Simplified Chinese,
// Traditional Chinese, Korean, Japanese, Russian, French.

func init() {
	extraTexts[xKimiTab] = [7]string{
		"Kimi Code", "Kimi Code", "Kimi Code", "Kimi Code", "Kimi Code", "Kimi Code", "Kimi Code",
	}
	extraTexts[xKimiTitle] = [7]string{
		"Quota", "额度", "額度", "할당량", "クォータ", "Квота", "Quota",
	}
	extraTexts[xKimiQuotaTitle] = [7]string{
		"Quota windows", "额度窗口", "額度視窗", "할당량 구간", "クォータ期間", "Окна квоты", "Fenêtres de quota",
	}
	extraTexts[xKimiLoading] = [7]string{
		"Loading", "正在读取", "正在讀取", "불러오는 중", "読み込み中", "Загрузка", "Chargement",
	}
	extraTexts[xKimiFetched] = [7]string{
		"Reading %@", "读取于 %@", "讀取於 %@", "읽은 시각 %@", "取得時刻 %@", "Прочитано %@", "Relevé à %@",
	}
	extraTexts[xKimiEndpoint] = [7]string{
		"Endpoint %@", "接口 %@", "介面 %@", "엔드포인트 %@", "エンドポイント %@", "Эндпоинт %@", "Point d’accès %@",
	}
	extraTexts[xKimiRegion] = [7]string{
		"Region %@", "区域 %@", "區域 %@", "지역 %@", "リージョン %@", "Регион %@", "Région %@",
	}
	extraTexts[xKimiRegionMainland] = [7]string{
		"Mainland China", "中国大陆", "中國大陸", "중국 본토", "中国本土", "Материковый Китай", "Chine continentale",
	}
	extraTexts[xKimiRegionGlobal] = [7]string{
		"Global", "国际", "國際", "글로벌", "グローバル", "Международный", "International",
	}
	extraTexts[xKimiRegionUnknown] = [7]string{
		"Region unknown", "区域未知", "區域未知", "지역 알 수 없음", "リージョン不明", "Регион неизвестен", "Région inconnue",
	}
	extraTexts[xKimiCredentialValid] = [7]string{
		"Credential valid until %@", "凭据有效期至 %@", "憑據有效期至 %@", "자격 증명 만료 %@", "資格情報の有効期限 %@", "Учётные данные действительны до %@", "Identifiant valable jusqu’au %@",
	}
	extraTexts[xKimiCredentialExpired] = [7]string{
		"The credential expired. Sign in again with the Kimi Code CLI (/login).", "凭据已过期，请用 Kimi Code CLI 重新登录（/login）。", "憑據已過期，請用 Kimi Code CLI 重新登入（/login）。", "자격 증명이 만료되었습니다. Kimi Code CLI에서 다시 로그인하세요(/login).", "資格情報の有効期限が切れました。Kimi Code CLI で再度ログインしてください（/login）。", "Срок действия учётных данных истёк. Войдите снова через Kimi Code CLI (/login).", "L’identifiant a expiré. Reconnectez-vous avec le CLI Kimi Code (/login).",
	}
	extraTexts[xKimiCredentialNoExpiry] = [7]string{
		"The credential carries no expiry time.", "凭据未记录过期时间。", "憑據未記錄過期時間。", "자격 증명에 만료 시각이 없습니다.", "資格情報に有効期限の記載がありません。", "В учётных данных нет срока действия.", "L’identifiant n’indique pas d’expiration.",
	}
	extraTexts[xKimiNoCredential] = [7]string{
		"No Kimi Code credential", "没有 Kimi Code 凭据", "沒有 Kimi Code 憑據", "Kimi Code 자격 증명이 없습니다", "Kimi Code の資格情報がありません", "Нет учётных данных Kimi Code", "Aucun identifiant Kimi Code",
	}
	extraTexts[xKimiNoCredentialDetail] = [7]string{
		"Sign in once with the Kimi Code CLI (/login). This panel only reads the credential file and never shows its contents.", "请先用 Kimi Code CLI 登录一次（/login）。此面板只读取凭据文件，从不显示其内容。", "請先用 Kimi Code CLI 登入一次（/login）。此面板只讀取憑據檔案，從不顯示其內容。", "Kimi Code CLI에서 한 번 로그인하세요(/login). 이 패널은 자격 증명 파일을 읽기만 하며 내용을 표시하지 않습니다.", "Kimi Code CLI で一度ログインしてください（/login）。このパネルは資格情報ファイルを読むだけで、内容は表示しません。", "Войдите один раз через Kimi Code CLI (/login). Панель только читает файл учётных данных и не показывает его содержимое.", "Connectez-vous une fois avec le CLI Kimi Code (/login). Ce panneau lit seulement le fichier d’identifiants et n’en affiche jamais le contenu.",
	}
	extraTexts[xKimiUnauthorized] = [7]string{
		"Kimi Code rejected the credential", "Kimi Code 拒绝了该凭据", "Kimi Code 拒絕了該憑據", "Kimi Code가 자격 증명을 거부했습니다", "Kimi Code が資格情報を拒否しました", "Kimi Code отклонил учётные данные", "Kimi Code a refusé l’identifiant",
	}
	extraTexts[xKimiUnauthorizedDetail] = [7]string{
		"Sign in again with the Kimi Code CLI (/login). A rejected credential is not refreshed automatically.", "请用 Kimi Code CLI 重新登录（/login）。被拒绝的凭据不会自动刷新。", "請用 Kimi Code CLI 重新登入（/login）。被拒絕的憑據不會自動重新整理。", "Kimi Code CLI에서 다시 로그인하세요(/login). 거부된 자격 증명은 자동으로 갱신되지 않습니다.", "Kimi Code CLI で再度ログインしてください（/login）。拒否された資格情報は自動更新されません。", "Войдите снова через Kimi Code CLI (/login). Отклонённые учётные данные не обновляются автоматически.", "Reconnectez-vous avec le CLI Kimi Code (/login). Un identifiant refusé n’est pas rafraîchi automatiquement.",
	}
	extraTexts[xKimiFetchFailed] = [7]string{
		"The quota request failed", "额度请求失败", "額度請求失敗", "할당량 요청 실패", "クォータの取得に失敗しました", "Запрос квоты не удался", "La requête de quota a échoué",
	}
	extraTexts[xKimiErrNetwork] = [7]string{
		"The endpoint could not be reached.", "无法连接到接口。", "無法連線到介面。", "엔드포인트에 연결할 수 없습니다.", "エンドポイントに接続できません。", "Не удалось связаться с эндпоинтом.", "Le point d’accès est injoignable.",
	}
	extraTexts[xKimiErrResponse] = [7]string{
		"The endpoint answered in a form this panel cannot read.", "接口返回了此面板无法解析的内容。", "介面回傳了此面板無法解析的內容。", "엔드포인트가 이 패널이 읽을 수 없는 형식으로 응답했습니다.", "エンドポイントがこのパネルで解釈できない形式で応答しました。", "Эндпоинт ответил в виде, который панель не может прочитать.", "Le point d’accès a répondu dans une forme illisible pour ce panneau.",
	}
	extraTexts[xKimiErrStatus] = [7]string{
		"The endpoint answered HTTP %@.", "接口返回 HTTP %@。", "介面回傳 HTTP %@。", "엔드포인트가 HTTP %@를 반환했습니다.", "エンドポイントが HTTP %@ を返しました。", "Эндпоинт вернул HTTP %@.", "Le point d’accès a répondu HTTP %@.",
	}
	extraTexts[xKimiErrCredential] = [7]string{
		"The credential file could not be read.", "无法读取凭据文件。", "無法讀取憑據檔案。", "자격 증명 파일을 읽을 수 없습니다.", "資格情報ファイルを読めません。", "Файл учётных данных не прочитан.", "Le fichier d’identifiants est illisible.",
	}
	extraTexts[xKimiErrUnknown] = [7]string{
		"The request failed.", "请求失败。", "請求失敗。", "요청이 실패했습니다.", "リクエストに失敗しました。", "Запрос не удался.", "La requête a échoué.",
	}
	extraTexts[xKimiLimit5h] = [7]string{
		"5-hour window", "5 小时窗口", "5 小時視窗", "5시간 구간", "5 時間枠", "окно 5 часов", "Fenêtre de 5 heures",
	}
	extraTexts[xKimiLimit7d] = [7]string{
		"7-day window", "7 天窗口", "7 天視窗", "7일 구간", "7 日枠", "окно 7 дней", "Fenêtre de 7 jours",
	}
	extraTexts[xKimiLimitMonthTotal] = [7]string{
		"Month total", "自然月总额", "自然月總額", "월 총액", "月間合計", "месячный итог", "Total du mois",
	}
	extraTexts[xKimiLimitMonthCode] = [7]string{
		"Month code", "自然月编程额度", "自然月編程額度", "월 코드 할당량", "月間コード枠", "месячная квота кода", "Quota code du mois",
	}
	extraTexts[xKimiUsed] = [7]string{
		"%@% used", "已用 %@%", "已用 %@%", "%@% 사용", "%@% 使用", "использовано %@%", "%@ % utilisé",
	}
	extraTexts[xKimiUnknown] = [7]string{
		"Unknown", "未知", "未知", "알 수 없음", "不明", "неизвестно", "Inconnu",
	}
	extraTexts[xKimiResetMissing] = [7]string{
		"The answer gave no reset time.", "响应未提供重置时间。", "回應未提供重置時間。", "응답에 재설정 시각이 없습니다.", "応答にリセット時刻がありません。", "В ответе нет времени сброса.", "La réponse n’indique pas d’heure de réinitialisation.",
	}
	extraTexts[xKimiQuotaNote] = [7]string{
		"A window the answer omits is shown as unknown, not as 0%.", "响应缺少的窗口显示为未知，而不是 0%。", "回應缺少的視窗顯示為未知，而不是 0%。", "응답에 없는 구간은 0%가 아니라 알 수 없음으로 표시됩니다.", "応答にない枠は 0% ではなく不明と表示します。", "Отсутствующее в ответе окно показано как неизвестное, а не 0%.", "Une fenêtre absente de la réponse est affichée comme inconnue, pas comme 0 %.",
	}
	extraTexts[xKimiWalletTitle] = [7]string{
		"Booster wallet", "加油包钱包", "加油包錢包", "부스터 지갑", "ブースターウォレット", "Кошелёк бустера", "Portefeuille booster",
	}
	extraTexts[xKimiWalletBalance] = [7]string{
		"Remaining %@ of %@", "剩余 %@，共 %@", "剩餘 %@，共 %@", "잔액 %@ / %@", "残り %@ / %@", "остаток %@ из %@", "Reste %@ sur %@",
	}
	extraTexts[xKimiWalletLimit] = [7]string{
		"Monthly charge limit %@, used %@", "月度扣费上限 %@，已用 %@", "月度扣費上限 %@，已用 %@", "월 청구 한도 %@, 사용 %@", "月額上限 %@、使用 %@", "месячный лимит %@, использовано %@", "Plafond mensuel %@, utilisé %@",
	}
	extraTexts[xKimiWalletLimitOff] = [7]string{
		"The monthly charge limit is off.", "月度扣费上限已关闭。", "月度扣費上限已關閉。", "월 청구 한도가 꺼져 있습니다.", "月額上限はオフです。", "Месячный лимит выключен.", "Le plafond mensuel est désactivé.",
	}
	extraTexts[xKimiWalletMissing] = [7]string{
		"The answer carried no booster wallet.", "响应中没有加油包钱包。", "回應中沒有加油包錢包。", "응답에 부스터 지갑이 없습니다.", "応答にブースターウォレットがありません。", "В ответе нет кошелька бустера.", "La réponse ne contient pas de portefeuille booster.",
	}
	extraTexts[xKimiCostTitle] = [7]string{
		"API-equivalent cost", "API等价成本", "API等價成本", "API 환산 비용", "API相当コスト", "Эквивалентная стоимость API", "Coût équivalent API",
	}
	extraTexts[xKimiCostEmpty] = [7]string{
		"No usage records under %@ yet.", "%@ 下还没有用量记录。", "%@ 下還沒有用量記錄。", "%@ 아래에 아직 사용 기록이 없습니다.", "%@ に使用記録がまだありません。", "В %@ ещё нет записей об использовании.", "Aucun enregistrement d’usage sous %@ pour l’instant.",
	}
	extraTexts[xKimiCostTokens] = [7]string{
		"In %@ · Cache read %@ · Cache write %@ · Out %@", "输入 %@ · 缓存读 %@ · 缓存写 %@ · 输出 %@", "輸入 %@ · 快取讀 %@ · 快取寫 %@ · 輸出 %@", "입력 %@ · 캐시 읽기 %@ · 캐시 쓰기 %@ · 출력 %@", "入力 %@ · キャッシュ読取り %@ · キャッシュ書込み %@ · 出力 %@", "Вход %@ · Чтение кэша %@ · Запись кэша %@ · Выход %@", "Entrée %@ · Lecture cache %@ · Écriture cache %@ · Sortie %@",
	}
	extraTexts[xKimiCostMissingPrice] = [7]string{
		"No price", "缺价格", "缺價格", "가격 없음", "価格なし", "нет цены", "prix manquant",
	}
	extraTexts[xKimiCostNote] = [7]string{
		"Kimi Code bills a subscription and its CLI stores token counts without a price list, so a model without a known price is shown as No price.", "Kimi Code 按订阅计费，CLI 只存 token 计数、不带价目表，因此没有已知价格的模型显示为“缺价格”。", "Kimi Code 按訂閱計費，CLI 只存 token 計數、不帶價目表，因此沒有已知價格的模型顯示為「缺價格」。", "Kimi Code는 구독으로 과금되며 CLI는 가격표 없이 토큰 수만 저장합니다. 알려진 가격이 없는 모델은 ‘가격 없음’으로 표시합니다.", "Kimi Code はサブスクリプション課金で、CLI は価格表を持たずトークン数のみを記録します。価格が不明なモデルは「価格なし」と表示します。", "Kimi Code оплачивается подпиской, а CLI хранит только счётчики токенов без прайс-листа, поэтому модель без известной цены помечается как «нет цены».", "Kimi Code est facturé par abonnement et son CLI ne stocke que des compteurs de tokens sans grille tarifaire ; un modèle sans prix connu est marqué « prix manquant ».",
	}
	extraTexts[xKimiCostScanned] = [7]string{
		"Scanned %@ session logs · %@ usage records", "已扫描 %@ 个会话日志 · %@ 条用量记录", "已掃描 %@ 個工作階段日誌 · %@ 筆用量記錄", "세션 로그 %@개 · 사용 기록 %@건 스캔", "セッションログ %@ 件 · 使用記録 %@ 件を走査", "Прочитано журналов сессий: %@ · записей использования: %@", "Journaux de session lus : %@ · enregistrements d’usage : %@",
	}
	extraTexts[xKimiNotesTitle] = [7]string{
		"Reading", "读取范围", "讀取範圍", "읽기 범위", "読み取り範囲", "Что читается", "Ce qui est lu",
	}
	extraTexts[xKimiNotesCredential] = [7]string{
		"The Kimi Code CLI owns the credential file. This panel writes it back only when a refresh rotates the token, and it never shows a token.", "凭据文件由 Kimi Code CLI 拥有。此面板只在刷新轮换了 token 时写回，且从不显示 token。", "憑據檔案由 Kimi Code CLI 擁有。此面板只在重新整理輪換了 token 時寫回，且從不顯示 token。", "자격 증명 파일은 Kimi Code CLI가 소유합니다. 이 패널은 갱신으로 토큰이 교체된 경우에만 다시 쓰며 토큰을 표시하지 않습니다.", "資格情報ファイルは Kimi Code CLI が所有します。このパネルはトークンが更新でローテーションされたときだけ書き戻し、トークンは表示しません。", "Файл учётных данных принадлежит Kimi Code CLI. Панель записывает его только при ротации токена и никогда не показывает токен.", "Le fichier d’identifiants appartient au CLI Kimi Code. Ce panneau ne le réécrit que lorsqu’un rafraîchissement fait tourner le jeton, et n’affiche jamais de jeton.",
	}
	extraTexts[xKimiLiveNote] = [7]string{
		"Live Kimi Code endpoint, not a local snapshot.", "实时 Kimi Code 接口，不是本地快照。", "即時 Kimi Code 介面，不是本機快照。", "실시간 Kimi Code 엔드포인트이며 로컬 스냅샷이 아닙니다.", "Kimi Code の実エンドポイントで、ローカルスナップショットではありません。", "Живой эндпоинт Kimi Code, а не локальный снимок.", "Point d’accès Kimi Code en direct, pas un instantané local.",
	}
}

const (
	xKimiTab                textKey = "kimiTab"
	xKimiTitle              textKey = "kimiTitle"
	xKimiQuotaTitle         textKey = "kimiQuotaTitle"
	xKimiLoading            textKey = "kimiLoading"
	xKimiFetched            textKey = "kimiFetched"
	xKimiEndpoint           textKey = "kimiEndpoint"
	xKimiRegion             textKey = "kimiRegion"
	xKimiRegionMainland     textKey = "kimiRegionMainland"
	xKimiRegionGlobal       textKey = "kimiRegionGlobal"
	xKimiRegionUnknown      textKey = "kimiRegionUnknown"
	xKimiCredentialValid    textKey = "kimiCredentialValid"
	xKimiCredentialExpired  textKey = "kimiCredentialExpired"
	xKimiCredentialNoExpiry textKey = "kimiCredentialNoExpiry"
	xKimiNoCredential       textKey = "kimiNoCredential"
	xKimiNoCredentialDetail textKey = "kimiNoCredentialDetail"
	xKimiUnauthorized       textKey = "kimiUnauthorized"
	xKimiUnauthorizedDetail textKey = "kimiUnauthorizedDetail"
	xKimiFetchFailed        textKey = "kimiFetchFailed"
	xKimiErrNetwork         textKey = "kimiErrNetwork"
	xKimiErrResponse        textKey = "kimiErrResponse"
	xKimiErrStatus          textKey = "kimiErrStatus"
	xKimiErrCredential      textKey = "kimiErrCredential"
	xKimiErrUnknown         textKey = "kimiErrUnknown"
	xKimiLimit5h            textKey = "kimiLimit5h"
	xKimiLimit7d            textKey = "kimiLimit7d"
	xKimiLimitMonthTotal    textKey = "kimiLimitMonthTotal"
	xKimiLimitMonthCode     textKey = "kimiLimitMonthCode"
	xKimiUsed               textKey = "kimiUsed"
	xKimiUnknown            textKey = "kimiUnknown"
	xKimiResetMissing       textKey = "kimiResetMissing"
	xKimiQuotaNote          textKey = "kimiQuotaNote"
	xKimiWalletTitle        textKey = "kimiWalletTitle"
	xKimiWalletBalance      textKey = "kimiWalletBalance"
	xKimiWalletLimit        textKey = "kimiWalletLimit"
	xKimiWalletLimitOff     textKey = "kimiWalletLimitOff"
	xKimiWalletMissing      textKey = "kimiWalletMissing"
	xKimiCostTitle          textKey = "kimiCostTitle"
	xKimiCostEmpty          textKey = "kimiCostEmpty"
	xKimiCostTokens         textKey = "kimiCostTokens"
	xKimiCostMissingPrice   textKey = "kimiCostMissingPrice"
	xKimiCostNote           textKey = "kimiCostNote"
	xKimiCostScanned        textKey = "kimiCostScanned"
	xKimiNotesTitle         textKey = "kimiNotesTitle"
	xKimiNotesCredential    textKey = "kimiNotesCredential"
	xKimiLiveNote           textKey = "kimiLiveNote"
)
