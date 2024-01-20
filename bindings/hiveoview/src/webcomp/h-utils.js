//=========================================================
// HiveOT Utility functions
//==========================================================


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

