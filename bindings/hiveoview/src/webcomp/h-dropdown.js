/**
 * Dropdown web-component.
 * Features:
 * - no color styling
 * - set title for the default button element
 * - slot for overriding the default button element
 * - any container can be used for content
 * - various positions for the dropdown relative to the button (see below)
 * - auto closes when removing focus
 * - animated slide-in/slide-out using clip-path
 * - move menu left to prevent running over the right side of the window
 *     (button loses its mouse receiver area if the window overlaps)
 */


const template = document.createElement('template')
template.innerHTML = `
    <div class="dropdown-container">
       <div button class="button">
           <!-- named slot 'button' to replace the default button-->
           <slot  name="button" button-slot>
              <button>menu</button>
           </slot>
       </div>

       <div content class="content" tabindex="-1">
            <slot content-slot></slot>
       </div>
    </div>
    
  <style>
  
  /* center in the parent container */
  :host {
      display: inline-flex;
  }
  
  /*Class used to control display of the content, where :focus-within handles the exit*/
  
  .dropdown-container { 
    position: relative;
    /* center aligning dropdown to the left and right*/
    align-items: center;
    /* center aligning dropdown above and below*/
    justify-content: center;
    /* inline-flex limits the dropdown button width to actual button size
      so the dropdown position is correct when using position rightXyz */
    display: inline-flex; 
  }
  .button {
     /*z-index:1;*/
  }
  .content {
      position: absolute;
      /*z-index: -1;   !* stay out of the way when hidden *! */
      width: max-content;
      /*padding: 3px;*/
      overflow: hidden;
      word-wrap: unset;

/*instead of display none, use clip-path to hide the menu, along with z-index.
 * z-index is set to -1 so it won't be in the way of other elements when hidden.
 * see each of the position styles for the clip-path best suited for the placement.
 * clip-path is awesome! see also https://bennettfeely.com/clippy/
 */
    transition: all 300ms ease;
}
  .bottom {
    top: 100%;
    clip-path: inset(0 0 100% 0); 
  }
  
  .bottomright {
    top: 100%;
    right:0;
    clip-path: inset(0 0 100% 0); 
  }
  .bottomleft {
   top:100%;
   left:0;
    clip-path: inset(0 0 100% 0); 
  }
   /* place container left from the button
    (requires container align-items: center) 
   */
  .left {
    right:100%;
    clip-path: inset(0 0 0 100%); 
  }
  .leftbottom {
    top: 100%;
    right: 100%;
    clip-path: inset(0 0 100% 100%); 
  }
  .lefttop {
    bottom: 100%;
    right: 100%;
    clip-path: inset(100% 0 0 100%); 
  }

  .right {
    left:100%;
    /* hide right to left */
    clip-path: inset(0 100% 0 0); 
  }
  .content.rightbottom {
    top: 100%;
    left: 100%;
    clip-path: inset(0 100% 100% 0); 
  }
  .righttop {
    bottom: 100%;
    left: 100%;
    clip-path: inset(100% 100% 0 0); 
  }
  .top {
  bottom: 100%;
    /* hide bottom-to top */
    clip-path: inset(100% 0 0 0); 
}

  .topright {
    bottom: 100%;
    right:0;
    /* hide bottom-to top */
    clip-path: inset(100% 0 0 0); 
  }
  .topleft {
    bottom: 100%;
    left:0;
    /* hide bottom-to top */
    clip-path: inset(100% 0 0 0); 
  }
  
  
  /*place these after content-slot so it overrides the default (hidden) clip-path*/
  .content:focus-within {
  /* content remains visible while it has focus*/
      z-index: 10;
      display: flex;  
      opacity: 1;
      /* unclipped*/
      clip-path: inset(0 0 0 0); 
  }

  .content.show {
      z-index: 10;
      display: flex;
      opacity: 1;
      /* unclipped*/
      clip-path: inset(0 0 0 0); 
      max-width: 1000px;
  }
  
</style>  
`;

/**
 * # h-dropdown custom web component
 * Usage:
 *   <h-dropdown position="top|bottom|left|right|...">
 *     <button slot="button"></button>
 *     <div>...content... </div>
 *  </h-dropdown>
 *
 * @param show: show the dropdown on initial render
 * @param title: string with title to show in the dropdown button element
 * @param position: position of the dropdown relative to the button
 *  bottom: center below the button
 *  bottomleft: left aligned below the button
 *  bottomright: right aligned below the button
 *  top: center above the button
 *  topleft: left aligned above the button
 *  topright: right aligned above the button
 *  left: left of the button
 *  lefttop, leftbottom: above or below left of the button
 *  right: right of the button
 *  righttop, rightbottom: above or below right of the button
 *
 * @slot button: replaces the default button element
 * @slot (default): places the dropdown content.
 */
class HDropdown extends HTMLElement {

    constructor() {
        super();
        const shadowRoot = this.attachShadow({mode: "open"});
        shadowRoot.append(template.content.cloneNode(true));

        // we need the click event from the child in the button slot to show
        // either the default button or the provided button.
        this.elButton = shadowRoot.querySelector("[button]");
        this.elButtonSlot = this.elButton.children[0]
        this.elButtonSlot.addEventListener("click", this.toggleMenu.bind(this))
        // if we're about to click on the button then ignore the focus event that
        // is the result of this click, to prevent the menu from toggling to open.
        this.elButtonSlot.addEventListener("mousedown", (ev) => {
            console.log("mousedown  set ignoreblur")
            this.ignoreBlur = true
        })

        this.elContent = shadowRoot.querySelector("[content]");
        this.elContentSlot = shadowRoot.querySelector("[content-slot]");
        // the children of the content slot must have a focus stop (tabIndex -1)
        if (this.elContentSlot.assignedElements) {
            let contentChildren = this.elContentSlot.assignedElements()
            this.contentChild = contentChildren[0]
            this.contentChild.tabIndex = -1
        }
        // when losing focus in the dropdown, close it.
        this.elContent.addEventListener("blur", this.hideContent.bind(this))

        // this.position = "bottom"
        // credits: react-autocomplete. ignore the blur event
        this.ignoreBlur = false
    }

    static get observedAttributes() {
        return ["position", "title", "show"];
    }

    attributeChangedCallback(name, oldValue, newValue) {
        console.log("attributeChangedCallback: " + name + "=" + newValue);
        if (name === "position") {
            this.elContent.classList.remove(oldValue)
            this.elContent.classList.add(newValue)
        } else if (name === "title") {
            // set the title to the dropdown button
            let buttonChild;
            if (this.elButtonSlot.children.length > 0) {
                buttonChild = this.elButtonSlot.children[0];
            }
            // if there the button slot is used, then use the element in this slot
            if (this.elButtonSlot.assignedElements) {
                let buttonChildren = this.elButtonSlot.assignedElements()
                if (buttonChildren.length > 0) {
                    buttonChild = buttonChildren[0];
                }
            }
            if (buttonChild) {
                // buttonChild.textContent = newValue;
                buttonChild.innerText = newValue;
            }

        } else if (name === "show") {
            this.elContent.classList.add("show")
        }
        // console.log("height:" + this.style.getPropertyValue("height"));
    }

    connectedCallback() {
    }

    // correct the position of the content if it moves past the right of the window
    // This sets translateX on the content element to shift it within view.
    correctContentPosition() {
        // inspired by http://www.quirksmode.org/js/findpos.html

        let width = this.elContent.offsetWidth
        let contentBox = this.elContent.getBoundingClientRect()
        let offsetLeft = this.elContent.offsetLeft
        let parentEl = this.elContent.offsetParent
        while (parentEl) {
            offsetLeft += parentEl.offsetLeft
            parentEl = parentEl.offsetParent
        }
        let diffX = window.innerWidth - offsetLeft - width;
        if (diffX < 0) {
            this.elContent.style.transform = "translateX(" + diffX + "px)";
        } else {
            this.elContent.style.transform = ""
        }
    }

    // hideContent removes the 'show' class from the content.
    hideContent(ev) {
        // ignore the blur event if it is the result of clicking the toggle button,
        // otherwise the toggle button click will show it again after hiding the dropdown.

        if (this.ignoreBlur) {
            console.log("hidecontent ignored")
            return
        }
        // if focus is set to content then the content remains visible until focus leaves
        // bug: when clicking the button, again, the content is hidden and shown again
        // because the toggle thinks its closed ('show' was remove and we can't tell
        // that focus was on the content).
        // Ideally the blur event (which calls hideContent) only happens when 'focus-within'
        // is no longer active.
        console.log("hidecontent remove show")
        this.elContent.classList.remove("show")
    }


    // toggleMenu toggles the menu by adding/removing the 'show' class.
    // When clicking the toggle button while the menu is shown, the onfocusout
    // handler is also invoked (hideContent method). This must come *after* this
    // menu toggle is called, otherwise hideContent will remove 'show' and the toggle
    // will then add 'show' again.
    toggleMenu(ev) {
        console.log("on toggle and remove ignoreblur")
        // the show class controls menu visibility
        let isShown = this.elContent.classList.contains("show")
        if (isShown) {
            this.elContent.classList.remove("show")
            this.elButton.focus()  // remove focus from content so it hides
        } else {
            this.elContent.classList.add("show")
            this.elContent.focus();
            // show content to determine size
            this.correctContentPosition()
        }
        this.ignoreBlur = false
    }

}

customElements.define('h-dropdown', HDropdown)
