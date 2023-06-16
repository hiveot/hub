<!--Data driven dropdown menu  -->
<script lang="ts" context="module">
  import { Button, Dropdown, DropdownItem, DropdownDivider, Label } from "flowbite-svelte";
  import MenuIcon from "$icons/Menu.svelte";
  // import MenuIcon from 'svelte-materialdesign-icons/Menu.svelte';

  export type IMenuItem = {
    label?: string,
    href?: string,
    icon?: any,  // optional icon or other component
    attr?: any,
    onClick?: (ev: IMenuItem) => void
  }
</script>

<script lang="ts">


  export let menu: IMenuItem[] = [];
  //@param title of dropdown
  export let title = "";

  export let onClick: (item: IMenuItem) => void | undefined;
</script>

<Button color="alternative" class="border-0">
  {title}
  <MenuIcon />
</Button>

<!-- Margin 0 prevents CSS margin style warning for popper -->
<Dropdown class="typehead">
  {#each [...menu] as item}
    {#if item.label}
      <DropdownItem class="flex"
                    href={item.href}
                    on:click={()=>{
                            // debugger;
                            if (item.onClick) item.onClick(item);
                            if (onClick) onClick(item);
                          }}>
        <div class="flex mr-2 w-5 h-5 items-center">
          <svelte:component this={item.icon} {...item.attr} />
        </div>
        <Label class="whitespace-nowrap">{item.label}</Label>
      </DropdownItem>
    {:else}
      <DropdownDivider />
    {/if}
  {/each}

</Dropdown>
