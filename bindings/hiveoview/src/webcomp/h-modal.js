/**
 * Modal web-component.
 *
 * Features:
 * - open modal by setting show attribute on this modal element
 * - close by:
 *   1. removing show attribute from this element
 *   2. Press the ESC key (todo)
 *   3. click outside the modal
 *   4. client dispatching 'cancel' event Event('cancel',{bubbles:true})
 *
 * - display content from slot,
 * - or use hx-get trigger to load the content
 * - slide in and out when 'animate' attribute is set.
 * - placement: center (default), top, bottom, left, right (todo)
 *
 * Element attributes:
 *  @attr animate      animate the open and closing of the modal (see element styles)
 *  @attr show         show the modal
 *  @attr showclose    show the close icon button in the topright modal corner
 *  @attr shadow       show a box shadow around the content panel
 *
 * Element styles:
 *  --mask-background    background color of the mask
 *  --mask-opacity       transparancy of the mask 0-1
 */

const closeIconSvg = "M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"
const template = document.createElement('template')
template.innerHTML = `
<div class="modal">  
    
    <div id="modal-mask" class="modal-mask"></div>
    
    <div modalContentSel class="modal-content ">
    
        <button id="modal-close-button" closeIconSel title="close me"
                class="modal-close-button"
               >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="14px">
                <title>close</title>
                <path d=${closeIconSvg} />
            </svg>
        </button>
        
        <slot>modal empty slot content</slot>
    </div>
</div>

<style>
  
 .modal {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    display: none;
    
    flex-direction: column; 
    align-items: center;
    justify-content: center;
}
  
  /*if the component has a show attribute set then show modal and fade in*/
  :host([show]) .modal {
    display: flex;
  }
    /* if animation is enabled then fade out */
  :host([animate]) .modal {
    /*  fadeOut sets display:flex while animating, keeping the transition visible.*/
    animation-name: fadeOut;
    animation-duration: var(--animation-duration);
    animation-timing-function: ease;
  }

  /*if animation is enabled then fade in*/
  :host([show][animate]) .modal {
    animation-name: fadeIn;
    animation-duration: var(--animation-duration);
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
    z-index: 1;
    /*tabindex: -1;*/
    opacity: var(--mask-opacity);
    /*background-color: var(--mask-background);*/
    background-color: var(--pico-background-color);

  }


/*The close button is show if the 'showclose' attribute is set
 it is placed relative to the model content div
*/
.modal-close-button {
    display: none; /* hidden unless 'showclose' attr is set */
    position:absolute;
    z-index:4;
    top: 0px;
    right: 0px;
    height: 30px;
    width: 30px;
    align-items: center;
    justify-content: center;
    padding: 0px;
    border-radius: 15px;
    border: 1px solid gray;
    cursor: pointer;
    transition: top 0.5s, right 0.5s ease;
}
 /* show the close button if the h-modal element has the 'showclose' attr set*/
 :host([showclose]) .modal-close-button {
      display: flex;
  } 
  /*On show of the modal, move the close button in position*/
  :host([show]) .modal-close-button {
    transition: top 1.5s, right 1.5s ease;
    top: -10px;
    right: -10px;
  }
  
.modal-content {
    /*position the content relative for the close button and the z-index to work*/
    position: relative;
    align-items: center;
    justify-content: center;
    padding: var(--pico-form-element-spacing-vertical) var(--pico-form-element-spacing-horizontal);
    z-index: 3;
    display: flex;
    flex-direction: column;
    opacity: 1;
    background-color: var(--pico-background-color);
    border: solid 1px #999;
    border-radius: 8px;
  }
/*Show a box-shadow around the content if the modal has the 'shadow' attribute set */
:host([shadow]) .modal-content {
    box-shadow: 0px 5px 5px -3px rgba(0, 0, 0, 0.2),
       0px 8px 10px 1px rgba(0, 0, 0, 0.14),
       0px 3px 14px 2px rgba(0, 0, 0, 0.12);
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
/* the trick to keep showing while fading out is to include display flex in
 all keyframes except the last. Very nifty.
 Credits: https://stackoverflow.com/questions/25794276/apply-style-after-css-animation-without-js
 */
    0% {
        transform: translateY(-0%);
        opacity: 1;
        display: flex;  
    }
    99.99% { /* ... The properties of something you want to change ... */
        display: flex; 
    }
    100% {
        transform: translateY(-30%);
        opacity: 0;
        display: none;
    }
}
    
</style>
`

/**
 * # h-modal custom web component
 * Usage:
 *   <h-modal position="top|bottom|left|right|...">
 *     <div>...content... </div>
 *  </h-modal>
 *
 * @attribute show: show the dialog on initial render
 * @attribute position: position of the dialog
 *  center: center in the middle (default)
 *  top: the top half
 *  left: left half
 *  right: right half
 *  bottom: bottom half
 *
 * @slot: content of the dialog
 */
class HModal extends HTMLElement {

    constructor() {
        super();
        // shadowroot lets the slot content be replaced using hx-get, without it the modal handling div's would be replaced.
        const shadowRoot = this.attachShadow({mode: "open"});
        // clone the template to support multiple instances
        shadowRoot.append(template.content.cloneNode(true));

    }

    static get observedAttributes() {
        return ["show", "shadow"];
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("h-modal attributeChangedCallback: " + name + "=" + newValue);
        if (name === "show") {
            // handled through css
        } else if (name === "shadow") {
            // handled through css
        }
    }

    adoptedCallback() {
        // console.log("h-modal adoptedCallback")
    }

    connectedCallback() {
        // console.log("h-modal connectedCallback")

        this.modalEl = this.shadowRoot.querySelector(".modal")
        this.modalMaskEl = this.shadowRoot.getElementById("modal-mask")
        this.modalCloseEl = this.shadowRoot.getElementById("modal-close-button")
        this.modalContentEl = this.shadowRoot.querySelector(".modal-content")

        this.modalCloseEl.addEventListener("click", this.closeModal.bind(this))
        this.modalMaskEl.addEventListener("click", this.closeModal.bind(this))
        this.modalContentEl.addEventListener("close-modal", this.closeModal.bind(this))
        this.addEventListener("keydown", (ev) => {
            if (ev.key == "Escape") {
                ev.stopImmediatePropagation();
                this.closeModal();
            }
        })
    }

    closeModal() {
        this.removeAttribute("show")
    }

    disconnectedCallback() {
        // console.log("h-modal disconnectedCallback")
    }


}


customElements.define('h-modal', HModal)
