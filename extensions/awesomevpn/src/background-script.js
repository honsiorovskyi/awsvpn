const red = "#a00000"
const green = "#008000"
const grey = "#606060"
const yellow = "#b39200"
const white = "#ffffff"

async function getVPNStatus() {
    const r = await fetch("http://127.0.0.1:35001/status")
    const j = await r.json()
    return j.status
}

function tag(text, bg, fg) {
    browser.browserAction.setBadgeText({
        text: text
    })
    browser.browserAction.setBadgeBackgroundColor({
        color: bg
    })

    if (fg) {
        browser.browserAction.setBadgeTextColor({
            color: fg
        })
    }
}

async function handleBrowserAction() {
    try {
        const s = await getVPNStatus()

        switch (s) {
            case "connected":
            case "connecting":
            case "disconnecting":
                fetch("http://127.0.0.1:35001/disconnect")
                return
            case "disconnected":
                browser.tabs.create({
                    active: false,
                    url: "http://127.0.0.1:35001/connect?method=redirect"
                })
                return
            default:
                console.log("unknown status:", s)
                return
        }
    } catch (e) {
        console.log("error", e)
    }
}

async function monitor() {
    try {
        const s = await getVPNStatus()
        switch(s) {
            case "connected":
                tag("on", green)
                return
            case "disconnected":
                tag("off", grey)
                return
            case "connecting":
            case "disconnecting":
                tag("...", yellow, white)
                return
            default:
                tag("?", red)
                console.log("unknown status:", s)
                return
        }
    } catch (e) {
        tag("err", red)
        return
    }
}
   
function closeVPNTab(tabId, changeInfo, tab) {
    if (changeInfo.url === "http://127.0.0.1:35001/") {
        browser.tabs.remove(tabId);
    }
}

browser.tabs.onUpdated.addListener(closeVPNTab) 
browser.browserAction.onClicked.addListener(handleBrowserAction)
setInterval(monitor, 500)
