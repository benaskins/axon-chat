<script>
	import { marked } from 'marked';

	let { content = '' } = $props();

	// Configure marked for safe, clean output
	marked.setOptions({
		breaks: false,
		gfm: true,
	});

	// Normalize single newlines to paragraph breaks for LLM output
	// that uses \n between paragraphs instead of \n\n.
	// Preserves code blocks, list items, and actual double newlines.
	function normalizeContent(text) {
		if (!text) return '';
		// Split on code blocks to preserve them
		const parts = text.split(/(```[\s\S]*?```)/);
		return parts.map((part, i) => {
			// Odd indices are code blocks — don't touch
			if (i % 2 === 1) return part;
			// Convert single newlines to double (paragraph breaks)
			// but don't touch lines that are already double-spaced or list items
			return part.replace(/(?<!\n)\n(?!\n|[-*+] |[0-9]+\. |#|```|\|)/g, '\n\n');
		}).join('');
	}

	let html = $derived(marked.parse(normalizeContent(content)));
</script>

<div class="markdown">
	{@html html}
</div>

<style>
	.markdown :global(p) {
		margin: 0.75em 0;
	}

	.markdown :global(p:first-child) {
		margin-top: 0;
	}

	.markdown :global(p:last-child) {
		margin-bottom: 0;
	}

	.markdown :global(h1),
	.markdown :global(h2),
	.markdown :global(h3) {
		color: var(--text-primary);
		margin: 1em 0 0.5em;
		line-height: 1.3;
	}

	.markdown :global(h1:first-child),
	.markdown :global(h2:first-child),
	.markdown :global(h3:first-child) {
		margin-top: 0;
	}

	.markdown :global(h1) {
		font-size: 1.3em;
	}

	.markdown :global(h2) {
		font-size: 1.15em;
	}

	.markdown :global(h3) {
		font-size: 1.05em;
	}

	.markdown :global(strong) {
		font-weight: 600;
		color: var(--text-primary);
	}

	.markdown :global(em) {
		color: var(--text-secondary);
	}

	.markdown :global(ul),
	.markdown :global(ol) {
		padding-left: 1.5em;
		margin: 0.5em 0;
	}

	.markdown :global(li) {
		margin: 0.25em 0;
	}

	.markdown :global(code) {
		font-family: var(--font-mono);
		font-size: 0.9em;
		background: var(--bg-tertiary);
		padding: 2px 6px;
		border-radius: 4px;
	}

	.markdown :global(pre) {
		background: var(--bg-tertiary);
		padding: 16px;
		border-radius: 8px;
		overflow-x: auto;
		margin: 0.75em 0;
	}

	.markdown :global(pre code) {
		background: none;
		padding: 0;
		font-size: 0.85em;
		line-height: 1.5;
	}

	.markdown :global(blockquote) {
		border-left: 3px solid var(--accent);
		padding-left: 12px;
		color: var(--text-secondary);
		margin: 0.75em 0;
	}

	.markdown :global(a) {
		color: var(--accent);
		text-decoration: none;
	}

	.markdown :global(a:hover) {
		text-decoration: underline;
	}

	.markdown :global(hr) {
		border: none;
		border-top: 1px solid var(--border);
		margin: 1em 0;
	}

	.markdown :global(table) {
		border-collapse: collapse;
		margin: 0.75em 0;
		width: auto;
		overflow-x: auto;
		display: block;
	}

	.markdown :global(th),
	.markdown :global(td) {
		border: 1px solid var(--border);
		padding: 6px 12px;
		text-align: left;
	}

	.markdown :global(th) {
		background: var(--bg-tertiary);
		color: var(--text-primary);
		font-weight: 600;
	}

	.markdown :global(tr:nth-child(even)) {
		background: rgba(42, 42, 42, 0.3);
	}
</style>
