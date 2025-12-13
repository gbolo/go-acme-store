# Icon Customization Guide

The UI currently uses Flaticon's uicons icon font library. You can easily customize or change icons.

## Current Icons

The UI uses the following Flaticon icons:
1. **Header (Vault/Shield)** - SVG shield icon (inline)
2. **Refresh** - SVG circular arrow icon (inline)
3. **View Certificate** - `fi fi-rr-diploma` (Flaticon uicons)
4. **View Full Chain** - `fi fi-rr-link-alt` (Flaticon uicons)
5. **Copy to Clipboard** - `fi fi-rr-copy-alt` (Flaticon uicons)
6. **Favicon** - Same shield/vault SVG as header

## Using Flaticon Icons

### Current Setup: Flaticon uicons (Icon Font)

The UI is already configured to use Flaticon's uicons library. The CDN links are included in the HTML:

```html
<link rel='stylesheet' href='https://cdn-uicons.flaticon.com/2.6.0/uicons-regular-rounded/css/uicons-regular-rounded.css'>
<link rel='stylesheet' href='https://cdn-uicons.flaticon.com/2.6.0/uicons-bold-rounded/css/uicons-bold-rounded.css'>
```

To change an icon, simply replace the class name:

**Example:**
```html
<!-- Current -->
<i class="fi fi-rr-diploma"></i>

<!-- Change to another icon -->
<i class="fi fi-rr-file-certificate"></i>
```

**Available icon styles:**
- `fi-rr-*` - Regular Rounded icons
- `fi-br-*` - Bold Rounded icons
- `fi-rs-*` - Regular Straight icons
- `fi-bs-*` - Bold Straight icons
- `fi-sr-*` - Solid Rounded icons
- `fi-ss-*` - Solid Straight icons

Browse all available icons at: https://www.flaticon.com/uicons

### Option 1: Using Different Flaticon uicons (Easiest)

1. **Download Icons from Flaticon**
   - Visit https://www.flaticon.com/
   - Search for icons (e.g., "vault", "refresh", "document", "link", "copy")
   - Select the "Interface" icon pack or any pack you prefer
   - Download as SVG format
   - **Note**: Check Flaticon's license requirements for attribution

2. **Replace Inline SVG in HTML/JS**

   For example, to replace the header icon:
   
   a. Open your downloaded SVG file in a text editor
   b. Copy the SVG code (starting from `<svg>` to `</svg>`)
   c. Replace the existing SVG in `index.html` and `demo.html`

   **Before:**
   ```html
   <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
       <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
   </svg>
   ```

   **After (with your Flaticon SVG):**
   ```html
   <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
       <!-- Your Flaticon SVG path data here -->
   </svg>
   ```

   **Important**: Keep these attributes for proper styling:
   - For header icon: `fill="currentColor"`
   - For button icons: `fill="none" stroke="currentColor" stroke-width="2"`

### Option 2: Using Image Files

If you prefer to use PNG/SVG files instead of inline SVG:

1. **Create an icons directory**
   ```bash
   mkdir -p web/static/icons
   ```

2. **Download and save icons**
   - Download icons from Flaticon as PNG or SVG
   - Save them in `web/static/icons/`
   - Recommended names:
     - `vault.svg`
     - `refresh.svg`
     - `document.svg`
     - `link.svg`
     - `copy.svg`

3. **Update HTML to use image files**

   In `index.html`, replace:
   ```html
   <div class="header-icon">
       <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
           <path d="..."/>
       </svg>
   </div>
   ```

   With:
   ```html
   <div class="header-icon">
       <img src="/static/icons/vault.svg" alt="Vault Icon">
   </div>
   ```

4. **Update CSS for image icons**

   Add to `style.css`:
   ```css
   .header-icon img {
       width: 64px;
       height: 64px;
       filter: brightness(0) invert(1) drop-shadow(0 2px 10px rgba(0, 0, 0, 0.3));
   }

   .btn img, .refresh-btn img {
       width: 16px;
       height: 16px;
   }
   ```

## Icon Locations in Code

### Header Icon
- **File**: `web/static/index.html`
- **Line**: Around line 18-23
- **Also in**: `web/static/demo.html`

### Refresh Button Icon
- **File**: `web/static/index.html`
- **Line**: Around line 49-52
- **Also in**: `web/static/demo.html`

### Certificate Action Icons
- **File**: `web/static/app.js`
- **Function**: `createCertificateCard()`
- **Lines**: Around 100-120

### Copy Button Icon
- **File**: `web/static/app.js`
- **Functions**: `viewCertificate()` and `viewCertificateChain()`
- **Lines**: Around 180-190 and 200-210

## Recommended Flaticon Icons

Here are some suggested searches on Flaticon that would work well:

1. **Header/Main Icon**:
   - Search: "vault", "security", "shield", "certificate", "lock"
   - Recommended: Vault icon, security shield

2. **Refresh Button**:
   - Search: "refresh", "reload", "sync", "update"
   - Recommended: Circular arrows

3. **View Certificate**:
   - Search: "document", "file", "certificate", "paper"
   - Recommended: Document with lines

4. **View Chain**:
   - Search: "chain", "link", "connection"
   - Recommended: Chain links

5. **Copy to Clipboard**:
   - Search: "copy", "clipboard", "duplicate"
   - Recommended: Two overlapping documents

## Attribution

If using Flaticon icons with a free license, you may need to add attribution. Add this to your footer in `index.html`:

```html
<footer>
    <p>ACME Certificate Store</p>
    <p class="icon-attribution">
        Icons made by <a href="https://www.flaticon.com/authors/[author]" title="[author]">
        [author]</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a>
    </p>
</footer>
```

Add to `style.css`:
```css
.icon-attribution {
    font-size: 0.8em;
    opacity: 0.7;
    margin-top: 10px;
}

.icon-attribution a {
    color: var(--primary-color);
    text-decoration: none;
}
```

## Using Icon Libraries (Alternative)

Instead of Flaticon, you can also use free icon libraries:

### Font Awesome (Free)
```html
<!-- Add to head of index.html -->
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">

<!-- Use in HTML -->
<i class="fas fa-shield-alt"></i>
<i class="fas fa-sync-alt"></i>
<i class="fas fa-file-alt"></i>
<i class="fas fa-link"></i>
<i class="fas fa-copy"></i>
```

### Heroicons
```html
<!-- Already using inline SVG from Heroicons-style icons -->
<!-- Visit: https://heroicons.com/ for more options -->
```

### Bootstrap Icons
```html
<!-- Add to head -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.0/font/bootstrap-icons.css">

<!-- Use in HTML -->
<i class="bi bi-shield-lock"></i>
<i class="bi bi-arrow-clockwise"></i>
<i class="bi bi-file-earmark-text"></i>
<i class="bi bi-link-45deg"></i>
<i class="bi bi-clipboard"></i>
```

## Tips

1. **Consistent Style**: Choose all icons from the same pack/style for consistency
2. **Size**: Icons should be simple and recognizable at small sizes
3. **Color**: Current icons use `currentColor` to inherit text color - keep this for consistency
4. **Testing**: Test icons on both light and dark backgrounds
5. **Performance**: Inline SVG is faster than loading multiple image files
6. **Accessibility**: Always include descriptive `aria-label` or `title` attributes

## Example: Complete Icon Replacement

Here's a complete example of replacing the header icon with a custom SVG:

```html
<!-- Original -->
<div class="header-icon">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
    </svg>
</div>

<!-- Replaced with Flaticon Vault Icon (example) -->
<div class="header-icon">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" fill="currentColor">
        <!-- Paste your Flaticon SVG path data here -->
        <path d="M256 0C114.6 0 0 114.6 0 256s114.6 256 256 256s256-114.6 256-256S397.4 0 256 0z..."/>
    </svg>
</div>
```

Remember to replace the icons in both `index.html` and `demo.html`, as well as in the JavaScript functions in `app.js`!

