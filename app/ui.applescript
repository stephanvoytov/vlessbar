-- VLessBar UI
-- Launched by the engine binary with args: ResourcesDir, EnginePath, BundlePath
--
-- All dialogs go through System Events so they reliably come to the front
-- (dialogs shown by a background osascript process are often invisible).
-- The whole flow is wrapped so a failure shows a message instead of nothing.

property resPath : ""
property engPath : ""

on run argv
	if (count of argv) < 2 then error "usage: ui.applescript <resources> <engine> [bundle]"
	set resPath to item 1 of argv
	set engPath to item 2 of argv
	set bundlePath to ""
	try
		set bundlePath to item 3 of argv
	end try
	try
		my runMain(bundlePath)
	on error errMsg number errNum
		if errNum is not -128 then my notifyError("Ошибка интерфейса: " & errMsg)
	end try
end run

on runMain(bundlePath)
	set homeDir to do shell script "echo $HOME"
	set installedSystem to "/Applications/VLessBar.app"
	set installedUser to homeDir & "/Applications/VLessBar.app"

	-- Offer to self-install (unless already installed / user said "later").
	if bundlePath is not "" and bundlePath is not installedSystem and bundlePath is not installedUser then
		set marker to homeDir & "/.vlessbar/.noinstall"
		set noflag to do shell script "test -e " & quoted form of marker & " && echo 1 || echo 0"
		if noflag is "0" then
			set dlg to my askButtons("Установить VLessBar в папку Программы?", {"Позже", "Установить"}, 2)
			if button returned of dlg is "Установить" then
				my installApp(bundlePath, homeDir, installedSystem, installedUser)
				return
			else
				do shell script "/bin/mkdir -p " & quoted form of (homeDir & "/.vlessbar") & " && /usr/bin/touch " & quoted form of marker
			end if
		end if
	end if

	my menuLoop()
end runMain

on installApp(bundlePath, homeDir, installedSystem, installedUser)
	set dest to installedSystem
	try
		do shell script "/usr/bin/ditto " & quoted form of bundlePath & " " & quoted form of installedSystem with administrator privileges
	on error
		-- no admin rights: install into the user's own Applications folder
		set dest to installedUser
		do shell script "/bin/mkdir -p " & quoted form of (homeDir & "/Applications")
		do shell script "/usr/bin/ditto " & quoted form of bundlePath & " " & quoted form of dest
	end try
	do shell script "/usr/bin/xattr -dr com.apple.quarantine " & quoted form of dest & " 2>/dev/null; true"
	do shell script "/usr/bin/open " & quoted form of dest
end installApp

on menuLoop()
	repeat
		set stxt to my getState()
		set menuItems to {"Подключить", "Отключить", "Сменить сервер", "Обновить подписку", "Указать ссылку подписки", "Показать HWID", "Статус", "Закрыть"}
		set selList to my pick(menuItems, "Статус: " & stxt, "Подключить")
		if selList is false then exit repeat
		set actionName to item 1 of selList
		if actionName is "Подключить" then
			my doAction("up")
		else if actionName is "Отключить" then
			my doAction("down")
		else if actionName is "Сменить сервер" then
			try
				my changeServer()
			end try
		else if actionName is "Обновить подписку" then
			my doAction("sub-update")
		else if actionName is "Указать ссылку подписки" then
			try
				my addSubscription()
			end try
		else if actionName is "Показать HWID" then
			my notifyInfo(my runEng("hwid"))
		else if actionName is "Статус" then
			my notifyInfo(my runEng("status"))
		else if actionName is "Закрыть" then
			exit repeat
		end if
	end repeat
end menuLoop

on runEng(cmdStr)
	try
		return do shell script quoted form of engPath & " " & cmdStr
	on error errMsg
		return "ОШИБКА: " & errMsg
	end try
end runEng

on getState()
	try
		return do shell script quoted form of engPath & " gui-state"
	on error
		return "нет данных"
	end try
end getState

on doAction(cmdStr)
	set resTxt to my runEng(cmdStr)
	if resTxt starts with "ОШИБКА:" then
		my notifyError(resTxt)
	else
		tell application "System Events" to display notification resTxt with title "VLessBar"
	end if
end doAction

on addSubscription()
	set dlg to my askText("Вставьте ссылку подписки (https://...):", "")
	set u to text returned of dlg
	if u is "" then return
	my notifyInfo(my runEng("sub-add " & quoted form of u))
end addSubscription

on changeServer()
	set rawTxt to my runEng("gui-servers")
	if rawTxt starts with "ОШИБКА:" or rawTxt is "" then
		my notifyInfo("Нет серверов. Сначала обновите подписку.")
		return
	end if
	set names to {}
	set idxs to {}
	set AppleScript's text item delimiters to linefeed
	set theLines to text items of rawTxt
	repeat with ln in theLines
		set s to ln as text
		if s is not "" then
			set AppleScript's text item delimiters to tab
			set parts to text items of s
			if (count of parts) > 1 then
				set end of idxs to item 1 of parts
				set end of names to item 2 of parts
			end if
		end if
	end repeat
	set AppleScript's text item delimiters to ""
	if (count of names) is 0 then
		my notifyInfo("Нет серверов.")
		return
	end if
	set chosen to my pick(names, "Выберите сервер", item 1 of names)
	if chosen is false then return
	set chosenName to item 1 of chosen
	repeat with i from 1 to count of names
		if item i of names is chosenName then
			my doAction("set " & (item i of idxs))
			exit repeat
		end if
	end repeat
end changeServer

-- === dialog helpers (System Events so dialogs always come to the front) ===

on askButtons(promptText, btnList, defBtn)
	tell application "System Events"
		activate
		return display dialog promptText with title "VLessBar" buttons btnList default button defBtn
	end tell
end askButtons

on askText(promptText, defaultText)
	tell application "System Events"
		activate
		return display dialog promptText default answer defaultText with title "VLessBar" buttons {"Отмена", "OK"} default button 2
	end tell
end askText

on notifyInfo(txt)
	tell application "System Events"
		activate
		display dialog txt with title "VLessBar" buttons {"OK"} default button 1
	end tell
end notifyInfo

on notifyError(txt)
	tell application "System Events"
		activate
		display dialog txt with title "VLessBar" buttons {"OK"} default button 1 with icon caution
	end tell
end notifyError

on pick(itemList, promptText, defItem)
	tell application "System Events" to activate
	return choose from list itemList with title "VLessBar" with prompt promptText default items {defItem} OK button name "Выполнить" cancel button name "Закрыть" without multiple selections allowed
end pick
