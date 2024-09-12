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
<div id="h-modal" class="h-modal">  
    
    <div id="h-modal-mask" class="h-modal-mask"></div>

    <div id="h-modal-content" modalContentSel class="h-modal-content ">
        <button id="h-modal-close-button" closeIconSel title="close me"
            class="h-modal-close-button"
           >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="14px">
                <title>close</title>
                <path d=${closeIconSvg} />
            </svg>
        </button>
        <div class="h-modal-slot">
            <slot>modal empty slot content</slot>
        </div>
      
    </div>
   

</div>

<style>
  
 .h-modal {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    display: none;
    z-index:10;
    
    flex-direction: column; 
    align-items: center;
    justify-content: center;
}
  
  /*if the component has a show attribute set then show modal and fade in*/
  :host([show]) .h-modal {
    display: flex;
  }
    /* if animation is enabled then fade out */
  :host([animate]) .h-modal {
    /*  fadeOut sets display:flex while animating, keeping the transition visible.*/
    animation-name: fadeOut;
    animation-duration: var(--animation-duration);
    animation-timing-function: ease;
  }

  /*if animation is enabled then fade in*/
  :host([show][animate]) .h-modal {
    animation-name: fadeIn;
    animation-duration: var(--animation-duration);
    animation-timing-function: ease;
    }

  /* mask closing when clicking outside the modal content */
  .h-modal-mask {
    display: block;
    position: fixed;
    cursor: pointer;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    z-index: 3;  /*same as h-modal-content*/
    opacity: var(--mask-opacity);
    background-color: var(--mask-background);

  }


/*The close button is show if the 'showclose' attribute is set
 it is placed relative to the model content div
*/
.h-modal-close-button {
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
    border-radius: 16px;  /* round button */
    border: 1px solid gray;
    cursor: pointer;
    transition: top 0.5s, right 0.5s ease;
}
 /* show the close button if the h-modal element has the 'showclose' attr set*/
 :host([showclose]) .h-modal-close-button {
      display: flex;
  } 
  /*On show of the modal, move the close button in position*/
  :host([show]) .h-modal-close-button {
    transition: top 1.5s, right 1.5s ease;
    top: -10px;
    right: -10px;
    z-index:5; /*cover modal-content*/
  }
  
.h-modal-content {
    /*position the content relative for the close button and the z-index to work*/
    position: relative;
    z-index: 3; /* same as mask */
  }
  /*The slot has the border with rounded corners. This requires overflow
   set to hidden otherwise the corners are cut off.
   Unfortunately this will cut off the close button so the close button
   must be set outside the div with overflow hidden*/
.h-modal-slot {
    /*position: relative;*/
    align-items: center;
    justify-content: center;
    display: flex;
    overflow:hidden; /*don't cut off rounded corners*/
    flex-direction: column;
    opacity: 1;
    background-color: var(--pico-background-color);
    border: 1px solid var(--pico-secondary-border);

    border-radius: var(--pico-card-border-radius);
}

/*Show a box-shadow around the content if the modal has the 'shadow' attribute set */
:host([shadow]) .h-modal-content {

  /*box-shadow: 0px 4px 8px -2px rgba(9, 30, 66, 0.35),*/
  /*      0px 0px 0px 1px rgba(37,64,111,0.59);*/
   box-shadow: var(--pico-box-shadow);
    

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
        this.classList.add("shadow")

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

        this.modalEl = this.shadowRoot.getElementById("h-modal")
        this.modalMaskEl = this.shadowRoot.getElementById("h-modal-mask")
        this.modalCloseEl = this.shadowRoot.getElementById("h-modal-close-button")
        this.modalContentEl = this.shadowRoot.getElementById("h-modal-content")

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
