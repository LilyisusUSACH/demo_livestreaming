# 🎬 Investigación UI/UX — Plataformas de Live Streaming en USA

> Análisis comparativo de las 8 principales plataformas de Live TV en Estados Unidos, con foco en el diseño de players, selectores de canales y patrones de UI.

---

## 📊 Matriz Comparativa Rápida

| Plataforma | Tema / Paleta | Navegación | Selector de Canales | Player | Tipografía | Lo más destacable |
|:---|:---|:---|:---|:---|:---|:---|
| **YouTube TV** | Dark Gray `#0F0F0F` + Red `#FF0000` | Top Bar | Mini Guide overlay + EPG drag & drop | Custom (Video.js), pill buttons | `Roboto` + `YouTube Sans` | Multiview 4 streams, Key Plays |
| **Fubo TV** | Charcoal `#12151D` + Orange `#FF5A00` | Top Bar | Drawer lateral + Sports filter | Custom, línea naranja, latency indicator | `Inter` style | Miniplayer dockable, odds de apuestas |
| **Hulu Live TV** | Rich Black `#040405` + Green `#1CE783` | Top Bar | Quick bar horizontal + Last Channel button | Minimalista, "Jump to Live" | `Graphik` | SVOD + Live TV integrado |
| **Sling TV** | Dark Navy `#0A111E` + Blue `#00A3E0` | **Left Sidebar colapsable** | Mini guide overlay + filtros de tier | Custom, bandwidth controller | Rounded Sans-Serif propietaria | Filtro Orange/Blue tier en EPG |
| **Peacock TV** | Midnight Black `#0B0D12` + Multicolor NBC | Top Bar | Grid 24/7 + category tabs | Custom, barra multicolor | `Peacock Sans` (custom) | Canales 24/7 curados, Sports widgets |
| **Philo** | Ink `#0B0E18` + Coral `#F4A38A` | Top Bar | **Half-sheet overlay** (no bloquea el video) | Custom, scrubber coral, botón DVR integrado | Serif `Honey` + Sans `Roobert` | Paleta cálida, tipografía serif, half-sheet |
| **DirecTV Stream** | Navy/Black + Blue `#0066D6` | Top Bar / Left Slide | Mini Guide 1-2 líneas en overlay | Estilo cable box, Electric Blue scrubber | Bold Sans-Serif alto contraste | Parity con experiencia de cable box |
| **Pluto TV** | Pure Black `#000000` + Yellow `#FEF200` | Top Bar + EPG tabs | **Grid directo** (click = swap instantáneo) | Nitro DS, Yellow scrubber, auto-play | Tokenized Sans-Serif (Nitro DS) | Auto-play sin click, split-screen nativo |

---

## 🔍 Análisis Detallado por Plataforma

### 1. 🔴 YouTube TV

**Colores**: Dark Gray `#0F0F0F` + YouTube Red `#FF0000`

**Layout del player**:
- Player full viewport con controles auto-ocultables
- Mini Guide accesible con scroll/flecha sin salir del video
- EPG en pestaña "Live" con drag-and-drop de canales favoritos

**Navegación**: `Library` | `Home` | `Live` — Top Bar horizontal

**Selector de canales**:
- Mini Guide flotante sobre el video (translúcido)
- Barra de "últimos canales vistos"
- EPG personalizable con drag & drop

**Player Controls** (totalmente custom):
- Botones pill-shaped: Play/Pause, ±10s seek, CC, resolución hasta 4K
- Botón **"Jump to Live"** para sincronizar con el directo
- **Stats for Nerds** overlay
- **Key Plays** para deportes en tiempo real
- **Multiview**: hasta 4 streams simultáneos

**Tipografía**: `Roboto` (UI) + `YouTube Sans` (marca)

**Animaciones**: Fade suave de controles, carruseles horizontales fluidos, drag-and-drop en EPG

---

### 2. 🟠 Fubo TV

**Colores**: Charcoal `#12151D` + Fubo Orange `#FF5A00`

**Layout del player**:
- Full-screen con controles auto-ocultables
- **Miniplayer dockable**: el video se minimiza en corner mientras navegas el EPG

**Navegación**: `Home` | `Sports` | `Guide` | `My Stuff` | `Search` — Top Bar

**Selector de canales**:
- Drawer lateral deslizable (mini-guide)
- Filtros rápidos: Sports / News / Movies / Favorites / Date pickers

**Player Controls**:
- Línea de progreso en **naranja** (`#FF5A00`)
- Indicador de latencia en tiempo real
- Volume slider, CC, Miniplayer toggle, fullscreen
- Miniplayer dockable en corner del navegador

**Características únicas**:
- Live scores, rosters y **betting odds integrados** al overlay del player
- Sports-centric como identidad de marca

**Animaciones**: Zoom en thumbnails hover, drawer slide suave, línea de progreso animada en naranja

---

### 3. 🟢 Hulu Live TV

**Colores**: Rich Black `#040405` + Electric Green `#1CE783`

**Layout del player**:
- Full viewport
- Semi-transparent guide overlay (hamburger icon) o vista docked player+guide

**Navegación**: `Home` | `Series` | `Movies` | `Live` | `My Stuff` — Top Bar minimalista

**Selector de canales**:
- **Quick Channel Bar**: drawer horizontal de recientes en parte inferior
- Botón **"Last"**: un click para volver al canal anterior
- Filtros: Favorites / Sports / News / Movies / Kids / Local

**Player Controls** (custom minimalista):
- Iconos pill-shaped, ±10s skip, "Jump to Live", audio & CC selector
- Focus rings en verde eléctrico `#1CE783`

**Tipografía**: `Graphik` (família geométrica sans-serif, look editorial)

**Diferenciador**: Única plataforma que mezcla SVOD (Disney+, ESPN+, Hulu VOD) + Live TV seamlessly

**Animaciones**: Backdrop blur translúcido, glowing green focus states, resize suave del player al abrir guide

---

### 4. 🔵 Sling TV

**Colores**: Dark Navy `#0A111E` + Sky Blue `#00A3E0` + Orange alerts

**Layout del player**:
- Full-screen o docked mientras se navega el EPG

**Navegación**: **Left Sidebar colapsable** (único en el grupo) + `Home` | `Guide` | `On Demand` | `DVR` | `Search`

**Selector de canales**:
- Left sidebar + Mini Guide overlay durante playback
- Filtro de tier: **Sling Orange vs Sling Blue** (según suscripción)
- **Freestream**: canales gratis integrados en el guide pago

**Player Controls**:
- Scrubber en azul `#00A3E0`
- ±10s jumps, live sync status, CC
- **Bandwidth Controller**: menú gear para seleccionar calidad manual

**Tipografía**: Rounded sans-serif propietaria (optimizada para 10-foot TV distance)

**Diferenciador**: Sidebar izquierdo, filtros de tier de suscripción, control de ancho de banda en player

---

### 5. 🦚 Peacock TV

**Colores**: Midnight Black `#0B0D12` + Multi-color NBC (Yellow, Orange, Red, Purple, Blue, Green)

**Layout del player**:
- Full-screen player con auto-hide
- Sección "Channels" con grid EPG 24/7 estilo cable

**Navegación**: `Home` | `TV Shows` | `Movies` | `Sports` | `Channels` — Top Bar

**Selector de canales**:
- Horizontal channel cards + grid vertical de programación
- Canales 24/7 curados: The Office 24/7, SNL 24/7

**Player Controls**:
- Barra de progreso **multicolor**
- Keyboard shortcuts nativos: `Space`/`K` Play, `←`/`→` Seek, `F` Fullscreen, `C` CC
- **"Player Pulse"**: overlay de estadísticas en tiempo real para NFL, Premier League, Olympics

**Tipografía**: `Peacock Sans` — tipografía custom con terminales angulares inspiradas en el pico del pavo real

**Diferenciador**: Tipografía custom única, canales 24/7 curados, sports widgets integrados

---

### 6. 🪸 Philo

**Colores**: Ink `#0B0E18` + Cream `#F5EFE7` + Alpenglow Coral `#F4A38A`

**Layout del player**:
- Full-screen no intrusivo
- **Half-sheet overlay**: guía que aparece en la mitad inferior SIN interrumpir el video

**Navegación**: `Home` | `Guide` | `Saved` | `Search` — Top Bar minimalista

**Selector de canales**:
- **Half-sheet / Third-sheet guide overlay**: cubre solo parte inferior de pantalla
- Canales favoritos fijados en la parte superior del guide

**Player Controls** (minimalista custom):
- Línea de progreso delgada expandible
- Scrubber handle en **coral** `#F4A38A`
- ±10s skip, botón **"Save" (DVR)** integrado directamente en el player chrome
- CC, selector de calidad

**Tipografía**:
- Headlines: Serif `Honey` (look editorial cálido, único en el sector)
- UI: Sans-serif `Roobert`

**Diferenciador**: Estética más humana/cálida vs la frialdad tech de otras plataformas; tipografía serif en headers

**Animaciones**: Half-sheet slide suave, coral button glow, mobile-first bottom sheet

---

### 7. 📡 DirecTV Stream

**Colores**: Deep Navy/Black + Electric Blue `#0066D6` + White

**Layout del player**:
- Full-screen con "Picture-in-Guide" al abrir el EPG
- Experiencia diseñada para paridad con cable box físico

**Navegación**: Top Menu horizontal / Left slide-out sidebar: `Watch Now` | `Guide` | `My Library` | `On Demand`

**Selector de canales**:
- **Mini Guide Overlay 1-2 líneas**: overlay mínimo en la parte inferior (no interrumpe)
- Navegación por número de canal + logo

**Player Controls**:
- Electric Blue timeline scrubber
- "Jump to Live", Play/Pause/Rewind/FF estándar
- Popover menu de audio & subtítulos

**Tipografía**: Bold sans-serif alto contraste (TV-first legibility)

**Diferenciador**: Experiencia fiel al cable box tradicional, "Start at Last Channel" auto-launch

---

### 8. 🟡 Pluto TV

**Colores**: Pure Black `#000000` + Pluto Yellow `#FEF200` + White

**Layout del player**:
- **Split-screen por defecto**: Player arriba + EPG grid abajo (sin fullscreen inicial)
- Auto-play instantáneo al cargar la página (zero clicks to video)

**Navegación**: `Live TV` | `On Demand` | `Search` + Category tabs sobre el guide

**Selector de canales**:
- **Click directo en cualquier fila del grid** → swap instantáneo del video sin reload de página
- Categorías: Featured / Movies / Entertainment / News / Binge Watch / Sports

**Player Controls** (Nitro Design System):
- Yellow scrubber `#FEF200`
- "Watch from Start" button
- CC, quality toggle, fullscreen
- Auto-hide controls

**Tipografía**: Geometric sans-serif tokenizado (Nitro DS, mín. 22px para TV legibility)

**Diferenciador**: Único en auto-play zero-click, split-screen como default, totalmente gratuito (AVOD)

**Animaciones**: Yellow focus border en channel cards, horizontal/vertical grid scroll suave

---

## 🔑 Patrones Comunes Identificados

### 1. 🌑 Dark Mode Universal
> **100% de las plataformas** usan dark mode como default. Reduce el glare, ahorra batería y hace que el video destaque visualmente. Fondos entre Pure Black y Dark Navy.

### 2. 📺 EPG Grid Estándar
> Canal a la izquierda + timeline horizontal a la derecha = patrón universal. Siempre incluye filtros por categoría (Sports / News / Movies / Favorites).

### 3. 🎛️ Surfing No-Disruptivo (3 técnicas)
- **Mini Guide Overlay**: Drawer translúcente sobre el video full-screen (YouTube TV, Hulu, Fubo, DirecTV, Philo)
- **Split View / Docked Player**: Video arriba, EPG abajo (Pluto TV, Fubo Miniplayer)
- **Hover Previews**: Preview del canal al pasar el cursor sobre el tile

### 4. ⚡ Custom Player Controls Live-Specific
Elementos que NO existen en players VOD normales:
- **"Jump to Live"** button: sincroniza con el directo tras pausa/rewind
- **±10s Seek Shortcuts**: adelantar/retroceder en incrementos
- **Sports Widgets**: scores en tiempo real, stats, key plays integrados

### 5. 🎨 Design Systems Tokenizados
Todas las plataformas usan design systems propios (Nitro DS de Pluto, YouTube Sans, Graphik de Hulu) con compliance WCAG AA y escalado para web + mobile + 10-foot TV.

### 6. ⚡ Time-to-First-Frame < 2 segundos
Auto-play inmediato o last-channel-on-load como prioridad de producto.

---

## 💡 Implicaciones para Nuestro Proyecto "Zapping"

| Característica | Plataforma Referente | Aplicación en Zapping |
|:---|:---|:---|
| Sidebar izquierdo animado | **Sling TV** | ✅ Ya implementado (mejorar animación 3D) |
| Split-screen player + EPG | **Pluto TV** | 💡 Considerar como layout alternativo |
| Mini Guide overlay translúcente | **YouTube TV / Hulu** | 💡 Overlay de canales durante playback |
| Botón "Jump to Live" | **Todos** | 💡 Essential para HLS live streams |
| Color accent fuerte (1 color) | **Fubo Orange / Hulu Green / Pluto Yellow** | ✅ Definir accent color de Zapping |
| Half-sheet guide (no intrusivo) | **Philo** | 💡 Alternativa al sidebar full |
| Live latency indicator | **Fubo TV** | 💡 Mostrar latencia HLS en el player |
| Tipografía editorial serif | **Philo** | 💡 Para titles/channel names destacados |

---

*Investigación realizada el 27 de Julio 2026 — Plataformas: YouTube TV, Fubo TV, Hulu Live TV, Sling TV, Peacock TV, Philo, DirecTV Stream, Pluto TV*
