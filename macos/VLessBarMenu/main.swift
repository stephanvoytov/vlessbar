// VLessBarMenu — macOS menu bar UI for VLessBar (AppKit, target 10.13 High Sierra).
//
// Talks to the engine binary (same .app bundle) via its CLI. No sandbox, no
// network code here: every action is `VLessBar <cmd> [...]` run asynchronously,
// output/exit-code decide success vs error dialog.
//
// Build (CI / build-macos.sh):
//   swiftc -O -target x86_64-apple-macosx10.13 -static-stdlib main.swift -o VLessBarMenu

import AppKit

// MARK: - Engine bridge

final class Engine {
    let path: String

    init(path: String) {
        self.path = path
    }

    /// Runs the engine CLI; completion is always called on the main queue.
    /// `code` is the process exit status (0 = success).
    func run(_ args: [String], completion: @escaping (_ out: String, _ code: Int32) -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            let p = Process()
            p.executableURL = URL(fileURLWithPath: self.path)
            p.arguments = args
            let pipe = Pipe()
            p.standardOutput = pipe
            p.standardError = pipe
            var text = ""
            var code: Int32 = -1
            do {
                try p.run()
                let data = pipe.fileHandleForReading.readDataToEndOfFile()
                p.waitUntilExit()
                text = String(data: data, encoding: .utf8) ?? ""
                code = p.terminationStatus
            } catch {
                text = "не удалось запустить движок: \(error.localizedDescription)"
                code = -1
            }
            text = text.trimmingCharacters(in: .whitespacesAndNewlines)
            DispatchQueue.main.async { completion(text, code) }
        }
    }
}

// MARK: - App delegate

final class AppDelegate: NSObject, NSApplicationDelegate, NSMenuDelegate {
    var engine: Engine!
    let menu = NSMenu()

    var statusItem: NSStatusItem!
    var timer: Timer?

    // Parsed state
    var stateOn = false
    var stateName = ""
    var stateLanOn = false
    var cachedServers: [(idx: Int, name: String)] = []
    var coreVersion = ""
    var busy = false

    // Menu items we update
    var statusLineItem: NSMenuItem!
    var toggleItem: NSMenuItem!
    var serversItem: NSMenuItem!
    var lanItem: NSMenuItem!
    var aboutItem: NSMenuItem!
    var actionItems: [NSMenuItem] = []

    // MARK: lifecycle

    func applicationDidFinishLaunching(_ note: Notification) {
        NSApp.setActivationPolicy(.accessory)
        engine = Engine(path: enginePath())

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        statusItem.button?.image = makeIcon(connected: false)
        statusItem.button?.toolTip = "VLessBar"

        menu.delegate = self
        menu.autoenablesItems = false
        statusItem.menu = menu
        buildMenu()

        refreshState()
        refreshServers()
        engine.run(["versions"]) { [weak self] out, _ in
            for line in out.split(separator: "\n") {
                if line.hasPrefix("core=") {
                    self?.coreVersion = String(line.dropFirst(5))
                }
            }
            self?.updateMenuTitles()
        }

        timer = Timer.scheduledTimer(withTimeInterval: 3.0, repeats: true) { [weak self] _ in
            self?.refreshState()
            self?.refreshServers()
        }
        timer?.tolerance = 1.0
    }

    func enginePath() -> String {
        let args = CommandLine.arguments
        if args.count > 1 {
            return args[1]
        }
        let exe = Bundle.main.executablePath ?? args[0]
        let dir = (exe as NSString).deletingLastPathComponent
        return dir + "/VLessBar"
    }

    func appVersion() -> String {
        (Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String) ?? "0.4.0"
    }

    // MARK: menu construction

    func buildMenu() {
        statusLineItem = NSMenuItem(title: "Состояние: …", action: nil, keyEquivalent: "")
        statusLineItem.isEnabled = false
        menu.addItem(statusLineItem)

        toggleItem = addItem("Подключить", #selector(toggleTapped))
        serversItem = NSMenuItem(title: "Серверы", action: nil, keyEquivalent: "")
        serversItem.submenu = NSMenu()
        menu.addItem(serversItem)

        menu.addItem(.separator())
        actionItems.append(addItem("Обновить подписку", #selector(subUpdateTapped)))
        actionItems.append(addItem("Ввести ссылку подписки…", #selector(addSubTapped)))

        menu.addItem(.separator())
        actionItems.append(addItem("Мой IP и пинг сервера", #selector(connTapped)))
        actionItems.append(addItem("Пинг через туннель", #selector(pingTunnelTapped)))
        lanItem = addItem("LAN-шлюз", #selector(lanTapped))
        actionItems.append(lanItem)

        menu.addItem(.separator())
        actionItems.append(addItem("Обновить приложение…", #selector(updateAppTapped)))
        actionItems.append(addItem("Показать HWID", #selector(hwidTapped)))

        aboutItem = NSMenuItem(title: "VLessBar", action: nil, keyEquivalent: "")
        aboutItem.isEnabled = false
        menu.addItem(aboutItem)

        menu.addItem(.separator())
        menu.addItem(addItem("Выход", #selector(quitTapped)))
    }

    func addItem(_ title: String, _ action: Selector) -> NSMenuItem {
        let it = NSMenuItem(title: title, action: action, keyEquivalent: "")
        it.target = self
        return it
    }

    func menuNeedsUpdate(_ m: NSMenu) {
        buildServersMenu() // synchronous, from cache
        updateMenuTitles()
    }

    func buildServersMenu() {
        let sub = NSMenu()
        sub.autoenablesItems = false
        if cachedServers.isEmpty {
            let it = NSMenuItem(title: "Нет серверов", action: nil, keyEquivalent: "")
            it.isEnabled = false
            sub.addItem(it)
        } else {
            for s in cachedServers {
                let it = NSMenuItem(title: s.name, action: #selector(serverPicked(_:)), keyEquivalent: "")
                it.target = self
                it.representedValue = s.idx
                it.state = (s.name == stateName) ? .on : .off
                it.isEnabled = !busy
                sub.addItem(it)
            }
        }
        serversItem.submenu = sub
    }

    func updateMenuTitles() {
        statusLineItem.title = "Состояние: \(stateOn ? "включено" : "выключено")  ·  Сервер: \(stateName.isEmpty ? "—" : stateName)"
        toggleItem.title = busy ? "Подождите…" : (stateOn ? "Отключить" : "Подключить")
        lanItem.state = stateLanOn ? .on : .off
        aboutItem.title = coreVersion.isEmpty ? "VLessBar \(appVersion())" : "VLessBar \(appVersion()) · Xray \(coreVersion)"
        for it in actionItems {
            it.isEnabled = !busy
        }
        toggleItem.isEnabled = !busy
    }

    // MARK: state

    func refreshState() {
        engine.run(["gui-state"]) { [weak self] out, code in
            guard let self = self else { return }
            if code == 0 {
                let parts = out.split(separator: "|").map { $0.trimmingCharacters(in: .whitespaces) }
                if parts.count > 0 {
                    self.stateOn = (parts[0] == "ON")
                }
                if parts.count > 1 {
                    self.stateName = (parts[1] == "(no server)") ? "" : parts[1]
                }
                self.stateLanOn = out.contains("lan: on")
            }
            self.statusItem.button?.image = self.makeIcon(connected: self.stateOn)
            self.updateMenuTitles()
        }
    }

    func refreshServers() {
        engine.run(["gui-servers"]) { [weak self] out, code in
            guard let self = self else { return }
            if code == 0 {
                var list: [(Int, String)] = []
                for line in out.split(separator: "\n") {
                    let f = line.split(separator: "\t", maxSplits: 1)
                    if f.count == 2, let i = Int(f[0]) {
                        list.append((i, String(f[1])))
                    }
                }
                self.cachedServers = list
            }
        }
    }

    /// Runs an engine command; on failure shows an alert, on success notifies.
    func runAction(_ args: [String], successMessage: String?, after: (() -> Void)? = nil) {
        busy = true
        updateMenuTitles()
        engine.run(args) { [weak self] out, code in
            guard let self = self else { return }
            self.busy = false
            if code != 0 {
                self.showAlert(out.isEmpty ? "ошибка" : out, error: true)
            } else if let msg = successMessage {
                self.notify(msg)
            }
            self.refreshState()
            self.refreshServers()
            after?()
        }
    }

    // MARK: actions

    @objc func toggleTapped() {
        guard !busy else { return }
        let connecting = !stateOn
        runAction([connecting ? "up" : "down"],
                  successMessage: connecting ? "Туннель подключён" : "Туннель отключён")
    }

    @objc func serverPicked(_ sender: NSMenuItem) {
        guard !busy, let idx = sender.representedValue as? Int else { return }
        runAction(["set", "\(idx)"], successMessage: "Сервер выбран")
    }

    @objc func subUpdateTapped() {
        runAction(["sub-update"], successMessage: "Подписка обновлена")
    }

    @objc func addSubTapped() {
        guard !busy else { return }
        NSApp.activate(ignoringOtherApps: true)
        let a = NSAlert()
        a.messageText = "Ссылка подписки"
        a.informativeText = "Вставьте https:// ссылку подписки:"
        let tf = NSTextField(frame: NSRect(x: 0, y: 0, width: 300, height: 24))
        a.accessoryView = tf
        a.addButton(withTitle: "OK")
        a.addButton(withTitle: "Отмена")
        NSApp.activate(ignoringOtherApps: true)
        guard a.runModal() == .alertFirstButtonReturn else { return }
        let url = tf.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !url.isEmpty else { return }
        runAction(["sub-add", url], successMessage: "Подписка добавлена")
    }

    @objc func connTapped() {
        guard !busy else { return }
        busy = true
        updateMenuTitles()
        engine.run(["conn"]) { [weak self] out, code in
            guard let self = self else { return }
            self.busy = false
            self.updateMenuTitles()
            if code != 0 && !out.contains("туннель") {
                self.showAlert(out, error: true)
            } else {
                self.showAlert(out)
            }
        }
    }

    @objc func pingTunnelTapped() {
        guard !busy else { return }
        busy = true
        updateMenuTitles()
        engine.run(["ping-tunnel"]) { [weak self] out, code in
            guard let self = self else { return }
            self.busy = false
            self.updateMenuTitles()
            if code != 0 {
                self.showAlert(out, error: true)
            } else {
                self.showAlert(out)
            }
        }
    }

    @objc func lanTapped() {
        guard !busy else { return }
        let next = stateLanOn ? "off" : "on"
        busy = true
        updateMenuTitles()
        engine.run(["lan", next]) { [weak self] out, code in
            guard let self = self else { return }
            self.busy = false
            if code != 0 {
                self.showAlert(out, error: true)
            } else {
                self.notify(self.stateLanOn ? "LAN выключен" : "LAN включён")
            }
            self.refreshState()
        }
    }

    @objc func hwidTapped() {
        guard !busy else { return }
        engine.run(["hwid"]) { [weak self] out, _ in
            self?.showAlert(out)
        }
    }

    @objc func updateAppTapped() {
        guard !busy else { return }
        busy = true
        updateMenuTitles()
        engine.run(["check-update"]) { [weak self] out, code in
            guard let self = self else { return }
            self.busy = false
            self.updateMenuTitles()
            if code != 0 {
                self.showAlert(out, error: true)
                return
            }
            if out.contains("url=") {
                NSApp.activate(ignoringOtherApps: true)
                let a = NSAlert()
                a.messageText = "Доступно обновление"
                a.informativeText = out
                a.addButton(withTitle: "Обновить")
                a.addButton(withTitle: "Отмена")
                if a.runModal() == .alertFirstButtonReturn {
                    self.engine.run(["update"]) { o2, c2 in
                        if c2 != 0 {
                            self.showAlert(o2, error: true)
                        } else {
                            self.notify("Обновление запущено")
                        }
                    }
                }
            } else {
                self.showAlert(out)
            }
        }
    }

    @objc func quitTapped() {
        NSApp.terminate(nil)
    }

    // MARK: feedback

    func notify(_ text: String) {
        let n = NSUserNotification()
        n.title = "VLessBar"
        n.informativeText = text
        NSUserNotificationCenter.default.deliver(n)
    }

    func showAlert(_ text: String, error isErr: Bool = false) {
        NSApp.activate(ignoringOtherApps: true)
        let a = NSAlert()
        a.messageText = "VLessBar"
        a.informativeText = text
        a.addButton(withTitle: "OK")
        if isErr {
            a.alertStyle = .warning
        }
        a.runModal()
    }

    // MARK: icon

    /// Globe template icon: outline when disconnected, filled when connected.
    func makeIcon(connected: Bool) -> NSImage {
        let img = NSImage(size: NSSize(width: 18, height: 18), flipped: false) { rect -> Bool in
            let outer = rect.insetBy(dx: 1, dy: 1)
            if connected {
                NSColor.black.setFill()
                NSBezierPath(ovalIn: outer).fill()
                NSColor.white.setStroke()
            } else {
                NSColor.black.setStroke()
            }
            let lw: CGFloat = connected ? 1.1 : 1.3
            // meridian
            let mer = NSBezierPath(ovalIn: NSRect(
                x: rect.midX - outer.width * 0.28,
                y: outer.minY,
                width: outer.width * 0.56,
                height: outer.height))
            mer.lineWidth = lw
            mer.stroke()
            // equator
            let eq = NSBezierPath()
            eq.move(to: NSPoint(x: outer.minX, y: rect.midY))
            eq.line(to: NSPoint(x: outer.maxX, y: rect.midY))
            eq.lineWidth = lw
            eq.stroke()
            if !connected {
                let outline = NSBezierPath(ovalIn: outer)
                outline.lineWidth = lw
                outline.stroke()
            }
            return true
        }
        img.isTemplate = true
        return img
    }
}

// MARK: - main

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
