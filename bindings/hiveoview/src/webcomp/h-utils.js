//=========================================================
// HiveOT Utility functions
//==========================================================
const IS_URL_TARGET_CLASS = "h-target"

/**
 * selectURLTargets sets the h-target class on the li parent of the element with an
 * href matching the current URL. Intended to highlight the navigation element of
 * the current URL.
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
window.selectURLTargets = (oldURL) => {
    let eList = document.documentElement.querySelectorAll(["[href]"])
    let newURL = window.location.href
    // console.log("href changed from ", oldURL, " to:", newURL, ". Found ", eList.length, "elements")
    eList.forEach((item) => {
        let liEl = item.closest('li')
        if (liEl) {
            // allow additional parameters after the URL
            if (newURL.startsWith(item.href)) {
                // item.classList.add(IS_URL_TARGET_CLASS)
                liEl.classList.add(IS_URL_TARGET_CLASS)
            } else if (oldURL.startsWith(item.href)) {
                // item.classList.remove(IS_URL_TARGET_CLASS)
                liEl.classList.remove(IS_URL_TARGET_CLASS)
            }
        }
    })
}

// listen for URL changes and re-activate the targets by setting the h-target class
window.addEventListener("hashchange", (ev) => {
    selectURLTargets(ev.oldURL)
})
selectURLTargets("")


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

