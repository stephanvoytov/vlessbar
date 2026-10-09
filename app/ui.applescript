-- VLessBar UI
-- Launched by the engine binary with args: ResourcesDir, EnginePath, BundlePath
-- Simple white/simple workflow: install, connect, disconnect, switch server.

property resPath : ""
property engPath : ""

on run argv
	set resPath to item 1 of argv
	set engPath to item 2 of argv
	try
		set bundlePath to item 3 of argv
	on error
		set bundlePath to ""
	end try

	-- Offer to self-install (unless already installed).
	set homeDir to do shell script "echo $HOME"
	set installedSystem to "/Applications/VLessBar.app"
	set installedUser to homeDir & "/Applications/VLessBar.app"
	if bundlePath is not "" and bundlePath is not installedSystem and bundlePath is not installedUser then
		set marker to homeDir & "/.vlessbar/.noinstall"
		set noflag to do shell script "test -e " & quoted form of marker & " && echo 1 || echo 0"
		if noflag is "0" then
			set dlg to display dialog "Установить VLessBar в «Программы»?" with title "VLessBar" buttons {"Позже", "Установить"} default button 2
			if button returned of dlg is "Установить" then
				set dest to installedSystem
				try
					do shell script "/usr/bin/ditto " & quoted form of bundlePath & " " & quoted form of installedSystem with administrator privileges
				on error
					-- no admin rights: install into the user's own Applications folder
					set dest to installedUser
					do shell script "/bin/mkdir -p " & quoted form of (homeDir & "/Applications")
					do shell script "/usr/bin/ditto " & quoted form of bundlePath & " " & quoted form of dest
				end try
				do shell script "/usr/bin/xattr -dr com.apple.quarantine " & quoted form of dest
				do shell script "/usr/bin/open " & quoted form of dest
				return
			else
				do shell script "/bin/mkdir -p " & quoted form of (homeDir & "/.vlessbar") & " && /usr/bin/touch " & quoted form of marker
			end if
		end if
	end if

	repeat
		set st to getState()
		set menuItems to {"Подключить", "Отключить", "Сменить сервер", "Обновить подписку", "Указать ссылку подписки", "Показать HWID", "Статус", "Закрыть"}
		set sel to choose from list menuItems with title "VLessBar" with prompt ("Статус: " & st) default items {"Подключить"} OK button name "Выполнить" cancel button name "Закрыть" without multiple selections allowed
		if sel is false then exit repeat
		set act to item 1 of sel
		if act is "Подключить" then
			doAction("up")
		else if act is "Отключить" then
			doAction("down")
		else if act is "Сменить сервер" then
			changeServer()
		else if act is "Обновить подписку" then
			doAction("sub-update")
		else if act is "Указать ссылку подписки" then
			addSubscription()
		else if act is "Показать HWID" then
			showInfo(runEng("hwid"))
		else if act is "Статус" then
			showInfo(runEng("status"))
		else if act is "Закрыть" then
			exit repeat
		end if
	end repeat
end run

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
	set res to runEng(cmdStr)
	if res starts with "ОШИБКА:" then
		display dialog res with title "VLessBar" buttons {"OK"} default button 1
	else
		display notification res with title "VLessBar"
	end if
end doAction

on addSubscription()
	set dlg to display dialog "Вставьте ссылку подписки (https://...):" default answer "" with title "VLessBar" buttons {"Отмена", "OK"} default button 2
	set u to text returned of dlg
	if u is "" then return
	set res to runEng("sub-add " & quoted form of u)
	display dialog res with title "VLessBar" buttons {"OK"} default button 1
end addSubscription

on changeServer()
	set raw to runEng("gui-servers")
	if raw starts with "ОШИБКА:" or raw is "" then
		display dialog "Нет серверов. Сначала обновите подписку." with title "VLessBar" buttons {"OK"} default button 1
		return
	end if
	set names to {}
	set idxs to {}
	set AppleScript's text item delimiters to linefeed
	set theLines to text items of raw
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
		display dialog "Нет серверов." with title "VLessBar" buttons {"OK"} default button 1
		return
	end if
	set chosen to choose from list names with title "VLessBar" with prompt "Выберите сервер" OK button name "Выбрать" cancel button name "Отмена" without multiple selections allowed
	if chosen is false then return
	set chosenName to item 1 of chosen
	repeat with i from 1 to count of names
		if item i of names is chosenName then
			doAction("set " & (item i of idxs))
			exit repeat
		end if
	end repeat
end changeServer

on showInfo(txt)
	display dialog txt with title "VLessBar" buttons {"OK"} default button 1
end showInfo
