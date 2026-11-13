package app

// OpenCode quota strings. The tab label is a product name, so it is the same
// in every language. English, Simplified Chinese, Traditional Chinese,
// Korean, Japanese, Russian, French — the same order as l10n.Languages.

func init() {
	extraTexts[xOpenCodeTab] = [7]string{
		"OpenCode", "OpenCode", "OpenCode", "OpenCode", "OpenCode", "OpenCode", "OpenCode",
	}
	extraTexts[xOpenCodeTitle] = [7]string{
		"OpenCode quota", "OpenCode 额度", "OpenCode 額度", "OpenCode 할당량", "OpenCode クォータ", "Квота OpenCode", "Quota OpenCode",
	}
	extraTexts[xOpenCodeConsole] = [7]string{
		"Console", "控制台", "主控台", "콘솔", "コンソール", "Консоль", "Console",
	}
	extraTexts[xOpenCodeRolling] = [7]string{
		"Rolling", "滚动窗口", "滾動視窗", "롤링", "ローリング", "Скользящее окно", "Fenêtre glissante",
	}
	extraTexts[xOpenCodeWeekly] = [7]string{
		"Weekly", "本周", "本週", "주간", "週間", "Неделя", "Semaine",
	}
	extraTexts[xOpenCodeMonthly] = [7]string{
		"Monthly", "本月", "本月", "월간", "月間", "Месяц", "Mois",
	}
	extraTexts[xOpenCodeUsed] = [7]string{
		"%@% used", "已用 %@%", "已用 %@%", "%@% 사용", "%@% 使用", "использовано %@%", "%@ % utilisé",
	}
	extraTexts[xOpenCodeLimited] = [7]string{
		"rate limited", "已限流", "已限流", "속도 제한", "レート制限中", "ограничено", "limité",
	}
	extraTexts[xOpenCodeNoData] = [7]string{
		"no data", "无数据", "無資料", "데이터 없음", "データなし", "нет данных", "aucune donnée",
	}
	extraTexts[xOpenCodeFetched] = [7]string{
		"Read at %@", "读取于 %@", "讀取於 %@", "%@에 읽음", "%@ に取得", "Прочитано %@", "Lu à %@",
	}
	extraTexts[xOpenCodeNoCredential] = [7]string{
		"Not signed in to OpenCode Console",
		"未登录 OpenCode 控制台",
		"未登入 OpenCode 主控台",
		"OpenCode 콘솔에 로그인되어 있지 않습니다",
		"OpenCode コンソールにサインインしていません",
		"Нет входа в консоль OpenCode",
		"Non connecté à la console OpenCode",
	}
	extraTexts[xOpenCodeNoCredentialDetail] = [7]string{
		"Sign in with the OpenCode CLI on this machine, then refresh. Quosparr reads the login the CLI stored and never writes to it.",
		"请在本机用 OpenCode CLI 登录后刷新。Quosparr 只读取 CLI 已保存的登录，绝不写入。",
		"請在本機用 OpenCode CLI 登入後重新整理。Quosparr 只讀取 CLI 已儲存的登入，絕不寫入。",
		"이 컴퓨터에서 OpenCode CLI로 로그인한 뒤 새로 고치세요. Quosparr는 CLI가 저장한 로그인을 읽기만 하고 쓰지 않습니다.",
		"このマシンで OpenCode CLI にサインインしてから更新してください。Quosparr は CLI が保存したログインを読むだけで、書き込みません。",
		"Войдите в OpenCode CLI на этой машине и обновите. Quosparr только читает сохранённый CLI вход и ничего не записывает.",
		"Connectez-vous avec le CLI OpenCode sur cette machine, puis actualisez. Quosparr lit la connexion enregistrée par le CLI et n’y écrit jamais.",
	}
	extraTexts[xOpenCodeExpired] = [7]string{
		"The OpenCode login has expired",
		"OpenCode 登录已过期",
		"OpenCode 登入已過期",
		"OpenCode 로그인이 만료되었습니다",
		"OpenCode のログインが期限切れです",
		"Срок входа в OpenCode истёк",
		"La connexion OpenCode a expiré",
	}
	extraTexts[xOpenCodeExpiredDetail] = [7]string{
		"Run the OpenCode CLI to sign in again. Quosparr does not refresh the token, because a second refresher would invalidate the CLI's own login.",
		"请重新运行 OpenCode CLI 登录。Quosparr 不刷新令牌，因为第二个刷新方会让 CLI 自己的登录失效。",
		"請重新執行 OpenCode CLI 登入。Quosparr 不重新整理權杖，因為第二個重新整理方會讓 CLI 自己的登入失效。",
		"다시 OpenCode CLI로 로그인하세요. Quosparr는 토큰을 갱신하지 않습니다. 다른 갱신 주체가 CLI의 로그인을 무효화하기 때문입니다.",
		"OpenCode CLI で再度サインインしてください。Quosparr はトークンを更新しません。別の更新主体が CLI のログインを無効にするためです。",
		"Войдите заново через OpenCode CLI. Quosparr не обновляет токен: второй обновляющий сделает вход CLI недействительным.",
		"Reconnectez-vous avec le CLI OpenCode. Quosparr ne renouvelle pas le jeton : un second renouvellement invaliderait la connexion du CLI.",
	}
	extraTexts[xOpenCodeError] = [7]string{
		"OpenCode usage is unavailable",
		"OpenCode 用量不可用",
		"OpenCode 用量無法取得",
		"OpenCode 사용량을 가져올 수 없습니다",
		"OpenCode の使用量を取得できません",
		"Использование OpenCode недоступно",
		"L’usage OpenCode est indisponible",
	}
	extraTexts[xOpenCodeUsageOff] = [7]string{
		"This organization has no OpenCode Go usage page.",
		"该组织没有 OpenCode Go 用量页。",
		"此組織沒有 OpenCode Go 用量頁。",
		"이 조직에는 OpenCode Go 사용량 페이지가 없습니다.",
		"この組織には OpenCode Go の使用量ページがありません。",
		"У этой организации нет страницы использования OpenCode Go.",
		"Cette organisation n’a pas de page d’usage OpenCode Go.",
	}
	extraTexts[xOpenCodeUsageFailed] = [7]string{
		"Go usage could not be read",
		"无法读取 Go 用量",
		"無法讀取 Go 用量",
		"Go 사용량을 읽을 수 없습니다",
		"Go 使用量を読み取れません",
		"Не удалось прочитать использование Go",
		"Impossible de lire l’usage Go",
	}
	extraTexts[xOpenCodeNoUsageData] = [7]string{
		"The answer carried no usage windows.",
		"响应中没有用量窗口。",
		"回應中沒有用量視窗。",
		"응답에 사용량 구간이 없습니다.",
		"応答に使用量の期間がありません。",
		"В ответе нет окон использования.",
		"La réponse ne contient aucune fenêtre d’usage.",
	}
	extraTexts[xOpenCodeBudgetTitle] = [7]string{
		"Console budget", "控制台预算", "主控台預算", "콘솔 예산", "コンソール予算", "Бюджет консоли", "Budget de la console",
	}
	extraTexts[xOpenCodeBudgetNone] = [7]string{
		"This organization has no budget rules.",
		"该组织没有预算规则。",
		"此組織沒有預算規則。",
		"이 조직에는 예산 규칙이 없습니다.",
		"この組織には予算ルールがありません。",
		"У этой организации нет правил бюджета.",
		"Cette organisation n’a aucune règle de budget.",
	}
	extraTexts[xOpenCodeBudgetDisabled] = [7]string{
		"Budgets are not enabled for this organization.",
		"该组织未开通预算功能。",
		"此組織未啟用預算功能。",
		"이 조직에서는 예산이 활성화되어 있지 않습니다.",
		"この組織では予算が有効になっていません。",
		"Бюджеты для этой организации не включены.",
		"Les budgets ne sont pas activés pour cette organisation.",
	}
	extraTexts[xOpenCodeBudgetSignedOut] = [7]string{
		"Sign in to read the Console budget.",
		"登录后才能读取控制台预算。",
		"登入後才能讀取主控台預算。",
		"로그인해야 콘솔 예산을 읽을 수 있습니다.",
		"サインインするとコンソール予算を読み取れます。",
		"Войдите, чтобы прочитать бюджет консоли.",
		"Connectez-vous pour lire le budget de la console.",
	}
	extraTexts[xOpenCodeBudgetFailed] = [7]string{
		"Budgets could not be read", "无法读取预算", "無法讀取預算", "예산을 읽을 수 없습니다",
		"予算を読み取れません", "Не удалось прочитать бюджеты", "Impossible de lire les budgets",
	}
	extraTexts[xOpenCodeBudgetSpent] = [7]string{
		"%@ of %@", "%@ / %@", "%@ / %@", "%@ / %@", "%@ / %@", "%@ из %@", "%@ sur %@",
	}
	extraTexts[xOpenCodeBudgetRule] = [7]string{
		"%@ · per %@", "%@ · 每 %@", "%@ · 每 %@", "%@ · %@마다", "%@ · %@ごと", "%@ · за %@", "%@ · par %@",
	}
	extraTexts[xOpenCodeCostTitle] = [7]string{
		"Recorded cost", "已记录成本", "已記錄成本", "기록된 비용", "記録されたコスト", "Записанная стоимость", "Coût enregistré",
	}
	extraTexts[xOpenCodeCostEmpty] = [7]string{
		"No priced OpenCode sessions on this machine yet.",
		"本机还没有记录成本的 OpenCode 会话。",
		"本機還沒有記錄成本的 OpenCode 工作階段。",
		"이 컴퓨터에는 아직 비용이 기록된 OpenCode 세션이 없습니다.",
		"このマシンにはまだコスト記録のある OpenCode セッションがありません。",
		"На этой машине пока нет сессий OpenCode с записанной стоимостью.",
		"Aucune session OpenCode facturée sur cette machine pour l’instant.",
	}
	extraTexts[xOpenCodeCostSessions] = [7]string{
		"%@ sessions", "%@ 个会话", "%@ 個工作階段", "세션 %@개", "%@ セッション", "сессий: %@", "%@ sessions",
	}
	extraTexts[xOpenCodeCostTokens] = [7]string{
		"In %@ · Out %@ · Reasoning %@ · Cache read %@ · Cache write %@",
		"输入 %@ · 输出 %@ · 推理 %@ · 缓存读 %@ · 缓存写 %@",
		"輸入 %@ · 輸出 %@ · 推理 %@ · 快取讀 %@ · 快取寫 %@",
		"입력 %@ · 출력 %@ · 추론 %@ · 캐시 읽기 %@ · 캐시 쓰기 %@",
		"入力 %@ · 出力 %@ · 推論 %@ · キャッシュ読取り %@ · キャッシュ書込み %@",
		"Вход %@ · Выход %@ · Рассуждения %@ · Чтение кэша %@ · Запись кэша %@",
		"Entrée %@ · Sortie %@ · Raisonnement %@ · Lecture cache %@ · Écriture cache %@",
	}
	extraTexts[xOpenCodeCostUntitled] = [7]string{
		"Sessions", "会话", "工作階段", "세션", "セッション", "Сессии", "Sessions",
	}
	extraTexts[xOpenCodeCostNote] = [7]string{
		"Cost is the USD amount the CLI recorded per session; this panel does not re-price tokens.",
		"成本是 CLI 按会话记录的美元金额；此面板不重算令牌价格。",
		"成本是 CLI 按工作階段記錄的美元金額；此面板不重算權杖價格。",
		"비용은 CLI가 세션마다 기록한 미국 달러 금액입니다. 이 패널은 토큰 가격을 다시 계산하지 않습니다.",
		"コストは CLI がセッションごとに記録した米ドル額です。このパネルはトークンを再計算しません。",
		"Стоимость — сумма в долларах, записанная CLI по сессиям; панель не пересчитывает токены.",
		"Le coût est le montant en dollars enregistré par le CLI par session ; ce panneau ne recalcule pas les jetons.",
	}
	extraTexts[xOpenCodeDatabaseFailed] = [7]string{
		"The local OpenCode database could not be read.",
		"无法读取本地 OpenCode 数据库。",
		"無法讀取本機 OpenCode 資料庫。",
		"로컬 OpenCode 데이터베이스를 읽을 수 없습니다.",
		"ローカルの OpenCode データベースを読み取れません。",
		"Локальная база OpenCode не прочитана.",
		"La base OpenCode locale est illisible.",
	}
	extraTexts[xOpenCodeSourceNote] = [7]string{
		"Read from the local OpenCode database (read-only) and the OpenCode usage endpoints.",
		"读取自本地 OpenCode 数据库（只读）与 OpenCode 用量接口。",
		"讀取自本機 OpenCode 資料庫（唯讀）與 OpenCode 用量介面。",
		"로컬 OpenCode 데이터베이스(읽기 전용)와 OpenCode 사용량 API에서 읽습니다.",
		"ローカルの OpenCode データベース（読み取り専用）と OpenCode 使用量 API から読み取ります。",
		"Читается из локальной базы OpenCode (только чтение) и API использования OpenCode.",
		"Lu depuis la base OpenCode locale (lecture seule) et les API d’usage OpenCode.",
	}
	extraTexts[xOpenCodeErrDecode] = [7]string{
		"The response could not be read.", "无法解析响应。", "無法解析回應。", "응답을 읽을 수 없습니다.",
		"応答を解釈できません。", "Ответ не прочитан.", "La réponse est illisible.",
	}
	extraTexts[xOpenCodeErrRequest] = [7]string{
		"The request failed.", "请求失败。", "請求失敗。", "요청이 실패했습니다.",
		"リクエストに失敗しました。", "Запрос не удался.", "La requête a échoué.",
	}
}

const (
	xOpenCodeTab                textKey = "opencodeTab"
	xOpenCodeTitle              textKey = "opencodeTitle"
	xOpenCodeConsole            textKey = "opencodeConsole"
	xOpenCodeRolling            textKey = "opencodeRolling"
	xOpenCodeWeekly             textKey = "opencodeWeekly"
	xOpenCodeMonthly            textKey = "opencodeMonthly"
	xOpenCodeUsed               textKey = "opencodeUsed"
	xOpenCodeLimited            textKey = "opencodeLimited"
	xOpenCodeNoData             textKey = "opencodeNoData"
	xOpenCodeFetched            textKey = "opencodeFetched"
	xOpenCodeNoCredential       textKey = "opencodeNoCredential"
	xOpenCodeNoCredentialDetail textKey = "opencodeNoCredentialDetail"
	xOpenCodeExpired            textKey = "opencodeExpired"
	xOpenCodeExpiredDetail      textKey = "opencodeExpiredDetail"
	xOpenCodeError              textKey = "opencodeError"
	xOpenCodeUsageOff           textKey = "opencodeUsageOff"
	xOpenCodeUsageFailed        textKey = "opencodeUsageFailed"
	xOpenCodeNoUsageData        textKey = "opencodeNoUsageData"
	xOpenCodeBudgetTitle        textKey = "opencodeBudgetTitle"
	xOpenCodeBudgetNone         textKey = "opencodeBudgetNone"
	xOpenCodeBudgetDisabled     textKey = "opencodeBudgetDisabled"
	xOpenCodeBudgetSignedOut    textKey = "opencodeBudgetSignedOut"
	xOpenCodeBudgetFailed       textKey = "opencodeBudgetFailed"
	xOpenCodeBudgetSpent        textKey = "opencodeBudgetSpent"
	xOpenCodeBudgetRule         textKey = "opencodeBudgetRule"
	xOpenCodeCostTitle          textKey = "opencodeCostTitle"
	xOpenCodeCostEmpty          textKey = "opencodeCostEmpty"
	xOpenCodeCostSessions       textKey = "opencodeCostSessions"
	xOpenCodeCostTokens         textKey = "opencodeCostTokens"
	xOpenCodeCostUntitled       textKey = "opencodeCostUntitled"
	xOpenCodeCostNote           textKey = "opencodeCostNote"
	xOpenCodeDatabaseFailed     textKey = "opencodeDatabaseFailed"
	xOpenCodeSourceNote         textKey = "opencodeSourceNote"
	xOpenCodeErrDecode          textKey = "opencodeErrDecode"
	xOpenCodeErrRequest         textKey = "opencodeErrRequest"
)
