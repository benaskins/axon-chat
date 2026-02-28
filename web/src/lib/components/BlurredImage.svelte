<script>
	let { src, isNsfw, alt = "Image" } = $props();
	let revealed = $state(false);

	function handleClick() {
		if (isNsfw && !revealed) {
			revealed = true;
		}
	}
</script>

<div class="image-wrapper" class:blurred={isNsfw && !revealed} onclick={handleClick}>
	<img {src} {alt} />
	{#if isNsfw && !revealed}
		<div class="blur-overlay">Click to view</div>
	{/if}
</div>

<style>
	.image-wrapper {
		position: relative;
		display: inline-block;
		width: 100%;
	}

	.image-wrapper img {
		width: 100%;
		height: auto;
		display: block;
	}

	.blurred img {
		filter: blur(20px);
		cursor: pointer;
	}

	.blur-overlay {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		background: rgba(0, 0, 0, 0.7);
		color: var(--text-primary);
		padding: 8px 16px;
		border-radius: 8px;
		pointer-events: none;
		font-size: 14px;
		white-space: nowrap;
	}
</style>
