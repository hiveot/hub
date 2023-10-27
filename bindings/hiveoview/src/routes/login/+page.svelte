<script lang="ts">
  import { enhance } from "$app/forms"; // magic???
  import { get } from "svelte/store";
  import { onMount, tick } from "svelte";
  import { Button, FloatingLabelInput, Heading, Label, Toggle } from "flowbite-svelte";

  /** @type {import("./$types").PageData} */
  export let data;
  /** @type {import("./$types").ActionData} */
  export let form;

  // reactive properties
  // start with an empty account record
  let rememberMe = data.rememberMe == "true";
  let password = "";
  $: isDisabled = (data.loginID == "" || password == "");

  // onMount(async () => {
  //   // load the form data after the parent has mounted to load account
  //   await tick();
  //   accountData = get(defaultAccount);
  // });

</script>

<div class="grid place-content-center h-full space-y-3">
  <!-- If no account is setup then ask for credentials -->
  <Heading tag="h4" class="">Login to the Hub</Heading>

  <form class="space-y-3 min-w-[400px]" method="POST"
        action="?/dologin"
        use:enhance>

    {#if form?.incorrect}<p class="error">Invalid password for user {data.loginID}</p>{/if}
    {#if form?.success}<p class="success">Login success as user {data.loginID}</p>
    {:else}
      <!-- Login ID and password input -->
      <FloatingLabelInput name="loginID" label="Email" type="text"
                          bind:value={data.loginID}
                          required
                          autocomplete="off"
      />
      <FloatingLabelInput name="password" label="Password" type="password"
                          bind:value={password}
                          required
                          autocomplete="off"
      />

      <!-- Remember the login info between sessions -->
      <div class="flex place-content-end space-x-2">
        <Label>Remember Me</Label>
        <Toggle name="rememberMe" bind:checked={rememberMe} />
      </div>

      <!-- Footer button -->
      <footer class="flex justify-between pt-5">
        <Button type="cancel" href="/">
          Cancel
        </Button>

        <Button type="submit" disabled={isDisabled}>
          Login
        </Button>
      </footer>
    {/if}

  </form>

</div>

