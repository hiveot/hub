//=========================================================
// HiveOT Utility functions
//==========================================================
const SELECT_IF_TARGET = "h-target"
const DISPLAY_IF_TARGET = "displayIfTarget"

// displayIfTarget sets the 'display' style on all elements that have the 'displayIfTarget'
// attribute. If the url path starts with the attribute value then set style display:flex,
// and set focus to the element, otherwise set the style display:none.
//
// This also works with query parameters, allowing the use of URL such as:
//   https://localhost:port/dashboard/page1 or dashboard?page1
// where displayIfTarget="/dashboard"
//
// newHash must begin with the # token, as must the displayIfTarget value.
window.displayIfTarget = (pathname) => {
    let eList = document.documentElement.querySelectorAll(["[" + DISPLAY_IF_TARGET + "]"])

    eList.forEach((item) => {
        let elVal = item.getAttribute(DISPLAY_IF_TARGET)
        if (elVal) {
            if (pathname.startsWith(elVal)) {
                item.style.setProperty('display', "flex")
                window.setTimeout(() => {
                    // focus on the item or group
                    if (item.tabIndex > 0) {
                        item.focus();
                    } else if (item.parentElement.tabIndex > 0) {
                        item.parentElement.focus()
                    }
                }, 0)
            } else {
                item.style.setProperty('display', "none")
            }
        }
    })
}

/**
 * selectNavTargets sets the h-target class on the li parent of the element with an
 * href matching the current URL. Intended to highlight the navigation element of
 * the current URL.
 *
 * TODO: is it better to use the 'selected' class instead?
 *
 * This uses querySelector to locate elements with an href, checks if it matching the
 * URL, and find the nearest parent li element.
 *
 * This is the equivalent of CSS li:has(a.h-target), which is not supported on older
 * browsers and mobile devices.
 *
 * Call this once on initialiation and each time the URL changes. (hashchange event)
 *
 * @param oldURL previous URL whose h-target to remove, or "" for the initial call.
 */
// set the 'h-target' class on elements with a href matching the URL
window.selectNavTargets = (oldURL, newURL) => {
    let eList = document.documentElement.querySelectorAll(["[href]"])
    // console.log("href changed from ", oldURL, " to:", newURL, ". Found ", eList.length, "elements")
    eList.forEach((item) => {
        let liEl = item.closest('li')
        if (liEl) {
            // allow additional parameters after the URL
            if (newURL.startsWith(item.href)) {
                // item.classList.add(SELECT_IF_TARGET)
                liEl.classList.add(SELECT_IF_TARGET)
            } else if (oldURL && (oldURL.startsWith(item.href))) {
                // item.classList.remove(SELECT_IF_TARGET)
                liEl.classList.remove(SELECT_IF_TARGET)
            }
        }
    })
}

// Navigate to a new URL without a browser reload.
// This:
// 1. stops the event propagation (otherwise htmx can throw an exception),
// 2. preventDefault to avoid a full page reload
// 3. pushes the new URL on the history
// 4. set the h-target class on the element matching the URL to highlight navigation element on the header menu
// 5. set the display style on elements with the displayIfTarget attribute of the URL to
//    show the newly selected page and set the focus to this element.
//
// This supports a fragment re-render of the given URL, so the given URL must have
// a server side renderer. When hx-boost is used there is no need for hx-get and hx-swap.
//
// Note that this does not trigger a popstate event.
//
// @param ev optional the event that triggered the request that will be stopped
// @param newURL is the destination URL
// This returns false to stop propagation in case of an onclick handler.
window.navigateTo = (ev, newURL) => {
    // FIXME: event is deprecated. how to stop propagation?
    if (ev) {
        ev.stopImmediatePropagation()
        ev.preventDefault()
    }
    let oldURL = location.href
    history.pushState("", "", newURL)
    selectNavTargets(oldURL, newURL)
    displayIfTarget(location.pathname)
    return false
}

/** Set the 'show' attribute on a HiveOT element
 * HiveOT elements can all be shown by adding the 'show' attribute and hidden
 * by removing this attribute.
 *
 * @param elementID ID of the element
 */
window.hShow = (elementID) => {
    let el = document.getElementById(elementID);
    if (el) {
        el.setAttribute("show", "");
    } else {
        console.error("hShow: element not found: ID=", elementID);
    }
}

// window.hShow = hShow

/** Remove the 'show' attribute from an HiveOT element
 *
 * @param elementID ID of the element
 */
window.hHide = (elementID) => {
    let el = document.getElementById(elementID);
    if (el) {
        el.removeAttribute("show");
    } else {
        console.error("hHide: element not found: ID=", elementID);
    }
}

