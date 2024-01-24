//=========================================================
// HiveOT Utility functions
//==========================================================
const IS_URL_TARGET_CLASS = "h-target"

/**
 * selectURLTargets sets the 'h-target' class on elements with a href matching the URL.
 * call this once on initialiation and each time the URL changes. (hashchange event)
 *
 * @param oldURL previous URL whose h-target to remove, or "" for the initial call.
 */
// set the 'h-target' class on elements with a href matching the URL
window.selectURLTargets = (oldURL) => {
    let eList = document.documentElement.querySelectorAll(["[href]"])
    let newURL = window.location.href
    console.log("href changed  to:", newURL)
    eList.forEach((item) => {
        // console.log("item: id=", item.id, ", href=", item.href)
        if (item.href === newURL) {
            item.classList.add(IS_URL_TARGET_CLASS)
        } else if (item.href === oldURL) {
            item.classList.remove(IS_URL_TARGET_CLASS)
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

