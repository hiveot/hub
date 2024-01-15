/**
 * Dialog web-component
 * Features:
 * - open by setting show attribute on this modal element
 * - close by:
 *   1. removing show attribute from this element
 *   2. Press the ESC key
 *   3. click the close icon in the top right, if enabled
 *   4. click the close button in the footer, if enabled
 *   5. click outside the modal
 *   Close should be treated as cancel.
 *
 * - show a close icon in the top right corner with the closeIcon attribute
 * - show a close button in the footer with the 'closeButton' attribute (when not using okCancel)
 * - show cancel and accept buttons in the footer when the 'okCancelButton' attribute is set
 * - slide in and out when 'animate' attribute is set. (todo)
 * - placement: center, top, bottom, left, right (todo)
 *
 * Custom attributes:
 * - showclose  show the close icon button in the topright modal corner
 */


const template = `
<div class="modal "
     _="on closeModal add .closing then wait for animationend then remove .closing from me remove @show from closest <h-modal2/> end " 
>
    <div class="modal-mask" _="on click trigger closeModal"></div>
    
    <div modalContentSel class="modal-content h-shadow"
         _="on keypress  log 'key pressed' "
    >
        <button closeIconSel title="close me"
                  _="on click trigger closeModal "
                  class="modal-close-button">
            <iconify-icon icon='mdi:close'></iconify-icon>
        </button>
            
        <div class="modal-header">
            <slot name="header"><h2>Title</h2></slot>
        </div>
    
        <div class="modal-body">
            <slot name="body"></slot>
        </div>
    
       <div class="modal-footer">
            <button closeButtonSel class="modal-button  uk-button "
                  _="on click trigger closeModal then send close">
               Close
            </button>
            <button cancelButtonSel class="modal-button  uk-button "
                         _="on click trigger closeModal then send cancel"
            >Cancel</button>
            
            <span style="flex-grow: 1"></span>
            <button submitButtonSel type="submit" class="button modal-button uk-button"
                         _="on click trigger closeModal then send submit"
            >Submit</button>
       </div>
  </div>
  </div>
  </div>
    
    
<style>
  
 .modal {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    
    /*The modal is hidden by default but still part of the DOM*/
    display: flex;
    flex-direction: column; 
    align-items: center;
    justify-content: center;
    
    /* Animate when opening */
    animation-name: fadeIn;
    animation-duration: 500ms;
    animation-timing-function: ease;
  }
  .modal.closing {
        /* Animate when closing */
        animation-name: fadeOut;
        animation-duration: 500ms;
        animation-timing-function: ease;
    }
    
    /* mask closing when clicking outside the modal content */
    .modal-mask {
        display: block;
        position: fixed;
        cursor: pointer;
        top: 0;
        left: 0;
        bottom: 0;
        right: 0;
        background: aliceblue;
        opacity: 80%;
        z-index: 1;
    }
    
  .modal-content {
        /*position relative for the close button and the z-index to work*/
        position: relative;
        padding: 10px;
        margin: 10px;
        z-index: 3;
        display: flex;
        flex-direction: column;
        opacity: 1;
        background: #e3e2e1;
        border: solid 1px #999;
        border-radius: 8px;

    }
        
  .modal-close-button {
    position:absolute;
    transition: top 1.5s, right 1.5s;
    /*top: -10px;*/
    /*right: -10px;*/
    z-index:4;
    top: 0px;
    right: 0px;
    height: 30px;
    width: 30px;
    padding: 0px;
    border-radius: 15px;
    border: 1px solid gray;
    display: none; /* hidden unless 'showclose' attr is set */
    align-items: center;
    justify-content: center;
  }
    /*withdraw the close button on closing*/
  .modal.closing .modal-close-button {
    top: 0px; 
    right: 0px; 
  }

  /* show the close button if the h-modal element has the 'showclose' attr set*/
  h-modal2[showclose] .modal-close-button {
      display: flex; 
  }
  
  /* move the close button into place when showing the view */
  h-modal2[show] .modal-close-button {
  /*FIXME: this only works on closing but not on opening */
     top: -10px; 
    right: -10px; 
  }
  
  .modal-header {
    position: relative;
    display: flex;
    flex-direction: row;
    justify-content: center;
  }
   
  /*Buttons are hidden unless enabled*/
  .modal-footer {
    display:flex;
    flex-direction: row;
    justify-content: center;
  }
  /*Control buttons are disabled unless explicitly enabled */
    .modal-button {
        display:none;
    }  
  
  
    .show {
        display:flex;
    }
  
    @keyframes fadeIn {
        0% {
            transform: translateY(-30%);
            opacity: 0;
        }
        100% {
            transform: translateY(-0%);
            opacity: 1;
        }
    }

    @keyframes fadeOut {
        0% {
            transform: translateY(-0%);
            opacity: 1;
        }
        100% {
            transform: translateY(-30%);
            opacity: 0;
        }
    }
    
</style>  
`;


/**
 * # h-modal custom web component
 * Usage:
 *   <h-modal position="top|bottom|left|right|...">
 *     <div>...content... </div>
 *  </h-dropdown>
 *
 * @param show: show the dialog on initial render
 * @param position: position of the dialog
 *  center: center in the middle (default)
 *  top: the top half
 *  left: left half
 *  right: right half
 *  bottom: bottom half
 *
 * @slot: content of the dialog
 */
class HModal2 extends HTMLElement {

    constructor() {
        super();
        // do not use the shadow dom by default
        // https://www.matuzo.at/blog/2023/pros-and-cons-of-shadow-dom/


        // activate the template so rendering doesn't show the slot content
        // this.innerHTML = template
        this.style.display = "none"
        this.onCancelCB = null
        this.onCloseCB = null
        this.onSubmitCB = null
        //
        this.childrenToSlots(template)
//
        this.modalContent = this.querySelector("[modalContentSel]")
        this.closeIconEl = this.querySelector("[closeIconSel]");
        this.cancelButtonEl = this.querySelector("[cancelButtonSel]");
        this.closeButtonEl = this.querySelector("[closeButtonSel]");
        this.submitButtonEl = this.querySelector("[submitButtonSel]");
        this.addEventListener("closeModal", (ev) => {
            console.log("on closeModel")
            // this.removeAttribute("show")
        })
    }

    static observedAttributes =
        ["position",
            "onclose", "oncancel", "onsubmit", "show"];


    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("h_modal2 attributeChangedCallback: " + name + "=" + newValue);

        if (name === "position") {
        } else if (name === "show") {
            this.style.display = (newValue === "" || newValue === null) ? "none" : "flex"
        } else if (name == "onclose") {
            this.onCloseCB = newValue
            this.closeButtonEl.style.display = "flex"
        } else if (name == "onsubmit") {
            this.onSubmitCB = newValue
            this.submitButtonEl.style.display = "flex"
        } else if (name == "oncancel") {
            this.onCancelCB = newValue
            this.cancelButtonEl.style.display = "flex"
        } else {
            console.log("unknown attribute: " + name)
            return
        }

    }

    // supports slots in light DOM
    // https://frontendmasters.com/blog/light-dom-only/
    childrenToSlots(html, src_elems) {
        var template = document.createElement("template");
        template.innerHTML = html;

        const slots = template.content.querySelectorAll("slot");
        for (const slot of slots) {
            const slotChildren = this.querySelectorAll(`[slot='${slot.name}']`);
            slot.replaceWith(...slotChildren);
        }

        this.replaceChildren(template.content);
    }

    connectedCallback() {
        this.onkeydown = (evt) => {
            console.log("onkey:", evt.key)
        }
        console.log("h-modal2 connectedCallback")
        // this.childrenToSlots(template)
    }

    disconnectedCallback() {
        console.log("h-modal2 disconnectedCallback")
    }

}

customElements.define('h-modal2', HModal2)
