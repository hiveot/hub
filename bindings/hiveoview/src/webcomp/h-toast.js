/* h-toast
 * This is a web element that shows a toast message when showToast is called.
 * Credits: https://www.codingnepalweb.com/toast-notification-html-css-javascript/
 *
 * Features:
 * - slide-in and slide-out animation on show and hide
 * - toast position; top or bottom and left or right
 * - default timeout and per-toast override
 * - toasts for info, success, error, and warning
 * - progress bar indicates remaining visible time
 * - halt countdown when hovering with the mouse (more time to read)
 *
 * Usage:
 *   html: <h-toast top|bottom left|right horizontal|vertical duration="1000"></h-toast>
 *   JS: window.toast.showToast(type, text, 3000)
 *
 * @attr top|bottom placement of the toast (default is top)
 * @attr left|right placement of the toast (default is center)
 * @attr horizontal|vertical animation (default is none)
 * @attr duration, the default time to show the toast
 *
 * @param type of toast: "info", "error", "success", "warning"
 * @param text to display
 * @param duration override default appearance duration with the given duration
 *
 * TODO:
 *  7. support htmx target innerhtml after
 *  8. support sse 'toast' {"type":"info","text","toast text"} events
 */
const PROGRESS_BAR_ANIMATION = "progressbar"

const template = `
<ul class="h-toast-container" >
</ul>

<style>
.h-toast-container{
    position: fixed;
    top:50px;     
    padding: 0;

    /* center the children */
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    
    z-index: 1;

  --dark: #34495E;
  --toast-background: var(--pico-background-color);
  --toast-border: var(--pico-border-color);
  --success: #0ABF30;
  --error: #E24D4C;
  --warning: #E9BD0C;
  --info: #3498DB;
  /*Duration is set in code*/
  --toast-duration: 5s;
    transition: all 800ms ease;  /* fixme animate collapsing multiples*/ 
}
h-toast[bottom] .h-toast-container {
    top: unset;
    bottom: 0px;
}
h-toast[left] .h-toast-container {
    left:20px;
    width: unset;
}
h-toast[right] .h-toast-container {
    right:20px;
    width: unset;
}

h-toast :where(.toast, .column) {
  display: flex;
  align-items: center;
}
h-toast .toast{
    width: fit-content;  
    position: relative;
    overflow:hidden;
    list-style: none;
    border-radius: 10px;
    padding: 16px;
    margin-bottom: 10px;
    background: var(--toast-background);
    justify-content: space-between;
    /*animation: show_toast_horizontal 1s ease forwards;*/
    --progress-ani-state: running
}
.toast:hover {
    animation-play-state: paused;
}
h-toast[horizontal] .toast{
    animation: show_toast_horizontal 1s ease forwards;
}
h-toast[vertical] .toast{
    animation: show_toast_vertical_top 1s ease forwards;
}
h-toast[vertical][bottom] .toast{
    animation: show_toast_vertical_bottom 1s ease forwards;
}

@keyframes show_toast_horizontal {
  0% {
    transform: translateX(-100%);
  }
  50% {
    transform: translateX(5%);
  }
  100% {
    transform: translateX(0%);
  }
}
@keyframes show_toast_vertical_top {
  0% {
    transform: translateY(-100%);
  }
  60% {
    transform: translateY(5%);
  }
  100% {
    transform: translateY(0%);
  }
}    
@keyframes show_toast_vertical_bottom {
  0% {
    transform: translateY(100%);
  }
  60% {
    transform: translateY(5%);
  }
  100% {
    transform: translateY(0px);
  }
}    

h-toast[horizontal]  .toast.hide {
    animation: hide_toast_horizontal 500ms ease forwards;
}
h-toast[vertical]  .toast.hide {
    animation: hide_toast_vertical 500ms ease forwards;
}
h-toast[vertical][bottom]  .toast.hide {
    animation: hide_toast_vertical_bottom 500ms ease forwards;
}
@keyframes hide_toast_horizontal {
  0% {
    transform: translateX(-10px);
  }
  100% {
    transform: translateX(200%);
  }
}
@keyframes hide_toast_vertical {
  0% {
    transform: translateY(-10px);
  }
  100% {
    transform: translateY(-200%);
  }
}
@keyframes hide_toast_vertical_bottom {
  0% {
    transform: translateY(0px);
  }
  100% {
    transform: translateY(200%);
  }
}

.toast::before {
  position: absolute;
  content: "";
  height: 2px;
  width: 100%;
  bottom: 0px;
  left: 0px;

  animation: ${PROGRESS_BAR_ANIMATION} var(--toast-duration) linear forwards;
  animation-play-state: var(--progress-ani-state);
}
.toast:hover {
  --progress-ani-state: paused;
}

@keyframes ${PROGRESS_BAR_ANIMATION} {
  100% {
    width: 0%;
  }
}

/*Toast border*/
.toast.error {
 border: 1px solid var(--error); 
}
.toast.info {
 border: 1px solid var(--info); 
}
.toast.success {
 border: 1px solid var(--success); 
}
.toast.warning {
 border: 1px solid var(--warning); 
}

/*Progress bar in color*/
.toast.success::before {
  background: var(--success);
}
.toast.error::before {
  background: var(--error);
}
.toast.warning::before {
  background: var(--warning);
}
.toast.info::before {
  background: var(--info);
}

/*Toast Icon color and size*/
.toast .column iconify-icon {
  font-size: 1.75rem;
}
.toast.error .column iconify-icon {
  color: var(--error);
}
.toast.info .column iconify-icon {
  color: var(--info);
}
.toast.success .column iconify-icon {
  color: var(--success);
}
.toast.warning .column iconify-icon {
  color: var(--warning);
}

.toast .column span {
  font-size: 1.07rem;
  margin-left: 12px;
}
.toast iconify-icon:last-child {
margin-left: 10px;
  color: #aeb0d7;
  cursor: pointer;
}
.toast iconify-icon:last-child:hover {
  color: var(--dark);
}

/*on small screen use up the full width*/
@media screen and (max-width: 530px) {
  .h-toast-container {
    width: 100%;
    padding-left: 20px;
    padding-right: 20px;
  }
  .h-toast-container .toast {
    width: 100%;
    font-size: 1rem;
    /*margin-left: 20px;*/
  }

}
</style>
`;

class HToast extends HTMLElement {

    // Object containing details for different types of toasts
    toastDetails = {
        success: {
            icon: 'mdi:check-circle',
            prefix: 'Success',
        },
        error: {
            icon: 'mdi:close-circle',
            prefix: 'Error',
        },
        warning: {
            icon: 'mdi:alert',
            prefix: 'Warning',
        },
        info: {
            icon: 'mdi:information',
            prefix: 'Info',
        }
    }

    static get observedAttributes() {
        return ["duration"]
    }

    constructor() {
        super();
        this.innerHTML = template;
        this.toastContainer = this.querySelector(".h-toast-container")
        this.duration = 5000;
        if (this.id) {
            window[this.id] = this
        }
        // window.toast1 = this
    }

    attributeChangedCallback(name, oldValue, newValue) {
        if (name === "duration") {
            this.duration = newValue;
            this.toastContainer.style.setProperty("--toast-duration", this.duration + "ms")
        }
    }

    connectedCallback() {
        this.toastContainer = this.querySelector(".h-toast-container")
    }

    /* Show a new toast of the given type (info, warning, error, success)
     * with an optional timeout.
     */
    showToast = (ttype, text, timeout) => {
        if (!timeout) {
            timeout = this.duration
        }
        // Getting the icon and text for the toast based on the id passed
        const {icon, prefix} = this.toastDetails[ttype];
        // Creating a new 'li' element for the toast
        const toast = document.createElement("li");
        toast.className = `toast ${ttype}`; // Setting the classes for the toast
        // timeout for countdown animation
        toast.style.setProperty("--toast-duration", timeout + "ms")

        // Setting the inner HTML for the toast
        toast.innerHTML =
            `<div class="column ${ttype}">
               <iconify-icon icon="${icon}"></iconify-icon>
               <span>${prefix}: ${text}</span>
             </div>
             <iconify-icon icon="mdi:close" onclick="parentElement.removeToast(this.parentElement)"></iconify-icon>
           `;
        toast.removeToast = this.removeToast
        this.toastContainer.appendChild(toast); // Append the toast to the notification ul
        // timeout to remove the toast after the progress animation ends
        toast.onanimationend = (ev) => {
            if (ev.animationName == PROGRESS_BAR_ANIMATION) {
                this.removeToast(toast);
            }
        }
    }

    removeToast = (toast) => {
        toast.classList.add("hide");
        if (toast.timeoutId) {
            clearTimeout(toast.timeoutId);
        } // Clearing the timeout for the toast
        // Removing the toast after 500ms
        // TODO: use timing from variables
        setTimeout(() => toast.remove(), 500);
    }
}


customElements.define('h-toast', HToast)
