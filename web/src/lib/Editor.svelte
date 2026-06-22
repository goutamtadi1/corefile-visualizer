<script>
  import { onMount, createEventDispatcher } from 'svelte'
  import { EditorView, keymap } from '@codemirror/view'
  import { defaultKeymap } from '@codemirror/commands'
  import { EditorState } from '@codemirror/state'

  export let value = ''
  const dispatch = createEventDispatcher()

  let host
  let view

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: value,
        extensions: [
          keymap.of(defaultKeymap),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) dispatch('change', u.state.doc.toString())
          }),
        ],
      }),
    })
    return () => view?.destroy()
  })

  function setDoc(text) {
    view?.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } })
  }

  async function onFile(e) {
    const file = e.target.files?.[0]
    if (!file) return
    const text = await file.text()
    setDoc(text)
    dispatch('change', text)
  }
</script>

<div class="editor">
  <label class="upload">
    Upload Corefile
    <input type="file" accept=".Corefile,.conf,text/plain" on:change={onFile} />
  </label>
  <div class="cm" bind:this={host} data-testid="editor"></div>
</div>
