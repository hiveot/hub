// Web element with Martin van Driel's awesome loading animation
// This shows the animation in the center of the view.
//
// Set a 'loading' class to change the settings below.
const template = `
<div>
    <div class="one"></div>
    <div class="two"></div>
    <div class="three"></div>
</div>   


<style>

    h-loading {
        position: absolute;
        top: calc(50% - 64px);
        left: calc(50% - 64px);
        width: 128px;
        height: 128px;
        border-radius: 50%;
        perspective: 800px;
    }

    h-loading div {
        position: absolute;
        box-sizing: border-box;
        width: 100%;
        height: 100%;
        border-radius: 50%;
    }

    h-loading .one {
        left: 0%;
        top: 0%;
        animation: rotate-one 1s linear infinite;
        border-bottom: 3px solid var(--pico-color);
    }

    h-loading .two {
        right: 0%;
        top: 0%;
        animation: rotate-two 1s linear infinite;
        border-right: 3px solid var(--pico-color);
    }

    h-loading .three {
        right: 0%;
        bottom: 0%;
        animation: rotate-three 1s linear infinite;
        border-top: 3px solid var(--pico-color);
    }


    @keyframes rotate-one {
        0% {
            transform: rotateX(35deg) rotateY(-45deg) rotateZ(0deg);
        }
        100% {
            transform: rotateX(35deg) rotateY(-45deg) rotateZ(360deg);
        }
    }

    @keyframes rotate-two {
        0% {
            transform: rotateX(50deg) rotateY(10deg) rotateZ(0deg);
        }
        100% {
            transform: rotateX(50deg) rotateY(10deg) rotateZ(360deg);
        }
    }

    @keyframes rotate-three {
        0% {
            transform: rotateX(35deg) rotateY(55deg) rotateZ(0deg);
        }
        100% {
            transform: rotateX(35deg) rotateY(55deg) rotateZ(360deg);
        }
    }
</style>

`


class HLoading extends HTMLElement {

    constructor() {
        super();
        this.innerHTML = template;
    }


    connectedCallback() {
    }
}

customElements.define('h-loading', HLoading)
