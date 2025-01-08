/** h-gridlayout is a simple webcomponent for placing widgets in a grid
 *
 * This places widgets 'cells' into a virtual grid and adds logic for:
 * 1. Preventing overlap of cells.
 * 2. Multiple breakpoints for layouts on different screen widths.
 * 3. Dragging of cells to new position
 * 4. Resizing cells
 * 5. Export and import layouts
 *
 * Usage:
 *  1. Create an instance of the grid component like any other html element.
 *  2. Provide a callback method for instantiating widgets based on the given cell ID.
 *     It is up to the application to store the grid layout and link cells to
 *     the widget to display.
 *  3. Optionally load the layout from a previously saved layout.
 *  4. Use 'addWidget' to add a new widget with the given location and size.
 *  5. Use 'removeWidget' to remove a widget with the given ID.
 *  6. Use 'moveWidget' to move the widget to a different grid position.
 *  7. Use 'resizeWidget' to reside a widget to a different grid size.
 *
 * Hooks for customizing behavior:
 *  1. getCellWidget retrieves the widget to place inside a cell.
 *  2. reflowHook implements the layout flow algorithm that adjusts the placement
 *     of elements in the layout when the position and size of a cell
 *     changes. A default algorithm plugin is included.
 *  3. getCellHeader retrieves the default header element in a widget.
 *     This allows customization of the widget banner or title based on the
 *     widget content. A default header is included.
 *  4. getMatrixSize hook supports custom algorithm to determine the size of
 *     matrix rows and columns based on the given view size. A default algorithm
 *     is included that provides the nr of columns for 4 breakpoints (lg,md,sm,xs).
 *
 * Placement and sizing:
 *   A view consists of matrix of columns and rows.
 *   Matrix rows have a fixed height. The default height is 100px;
 *   Matrix columns have an average of 100px width which can increase and decrease
 *   with the screen width.
 *
 *   Cell placement is in row and columns.
 *   done using cell x,y,w,h where each unit is a row or column nr
 *
 */

const template = document.createElement('template')
template.innerHTML = `
<div class="h-gridlayout" style="position:relative; width:100%; height:inherit">
</div>  
    
<style>
    .h-gridlayout {
        display: grid;
        gap: 3px;
        grid-template-columns: repeat(24,1fr);
    }
</style>
`
export class HGridLayout extends HTMLElement {

    static get observedAttributes() {
        return ["columns"]
    }

    constructor() {
        super();

        const shadowRoot = this.attachShadow({mode: "open"});
        shadowRoot.append(template.content.cloneNode(true));
    }


    attributeChangedCallback(name, oldValue, newValue) {
        if (name === "columns") {
            this.columns = newValue
        }
    }

    connectedCallback() {

    }


    disconnectedCallback() {

    }
}


// window.HChartJS = HTimechart
customElements.define('h-gridlayout', HGridLayout)

