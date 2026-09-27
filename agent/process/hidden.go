package process

// hidden liste les processus de la session Windows qui tournent sous le compte de
// l'utilisateur mais font partie du système : les fermer casserait le bureau.
var hidden = map[string]bool{
	"sihost.exe":                      true,
	"svchost.exe":                     true,
	"taskhostw.exe":                   true,
	"dwm.exe":                         true,
	"ctfmon.exe":                      true,
	"conhost.exe":                     true,
	"dllhost.exe":                     true,
	"runtimebroker.exe":               true,
	"fontdrvhost.exe":                 true,
	"startmenuexperiencehost.exe":     true,
	"shellexperiencehost.exe":         true,
	"searchhost.exe":                  true,
	"searchapp.exe":                   true,
	"textinputhost.exe":               true,
	"smartscreen.exe":                 true,
	"securityhealthsystray.exe":       true,
	"applicationframehost.exe":        true,
	"systemsettingsbroker.exe":        true,
	"backgroundtaskhost.exe":          true,
	"widgets.exe":                     true,
	"widgetservice.exe":               true,
	"lockapp.exe":                     true,
	"useroobebroker.exe":              true,
	"crossdeviceresume.exe":           true,
	"phoneexperiencehost.exe":         true,
	"windowspackagemanagerserver.exe": true,
}
