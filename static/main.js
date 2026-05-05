// main.js - client-side logic for Groupie Tracker
// Handles: live search with suggestions, artist card rendering, and dual range sliders.

// --- Search elements ---
const input = document.getElementById("search");
const suggestionsList = document.getElementById("suggestions");
const container = document.querySelector(".grid-container");
const bottomSection = document.querySelector(".pagination");
// Snapshot the server-rendered cards so we can restore them when the search is cleared
const initialCards = container.innerHTML;
const initialBottomDisplay = bottomSection ? bottomSection.style.display : "block";

// renderResults fetches search results from the server and updates the card grid.
// When the query is empty it restores the original server-rendered content.
async function renderResults(query) {
    if (!query.trim()) {
        suggestionsList.innerHTML = "";
        container.innerHTML = initialCards;
        if (bottomSection) bottomSection.style.display = initialBottomDisplay;
        return;
    }

    const res = await fetch(`/search?q=${encodeURIComponent(query)}`);
    const data = await res.json();

    container.innerHTML = "";
    // Hide pagination while showing live search results
    if (bottomSection) bottomSection.style.display = "none";

    // Build a card for each matching artist
    data.results.forEach(artist => {
        container.innerHTML += `
            <div class="card">
                <div class="card-content">
                    <div class="column">
                        <h2><a href="/artist/${artist.id}">${artist.name}</a></h2>
                        <a href="/artist/${artist.id}"><img src="${artist.image}" width="200" alt="${artist.name}"></a>
                        <p>First Album: ${artist.firstAlbum}</p>
                        <p>Creation Date: ${artist.creationDate}</p>
                    </div>
                    <div class="column">
                        <h4>Members:</h4>
                        <ul>
                            ${artist.members.map(m => `<li>${m}</li>`).join("")}
                        </ul>
                        <a href="/artist/${artist.id}?tab=dates"><p>Concert Dates</p></a>
                        <a href="/artist/${artist.id}?tab=locations"><p>Concert Locations</p></a>
                        <a href="/artist/${artist.id}?tab=relations"><p>Concert Dates And Locations</p></a>
                    </div>
                </div>
            </div>
        `;
    });

    // Render autocomplete suggestion items below the search input
    suggestionsList.innerHTML = "";
    data.suggestions.forEach(suggestion => {
        const item = document.createElement("li");
        item.className = "suggestion-item";
        item.innerHTML = `<span class="suggestion-text">${suggestion.text}</span><span class="suggestion-type">${suggestion.type}</span>`;
        // Clicking a suggestion fills the input and re-runs the search
        item.addEventListener("click", () => {
            input.value = suggestion.text;
            renderResults(suggestion.text);
            suggestionsList.innerHTML = "";
        });
        suggestionsList.appendChild(item);
    });
}

// Trigger a new search on every keystroke
input.addEventListener("input", () => {
    renderResults(input.value);
});

// Close the suggestions dropdown when clicking outside the search box
document.addEventListener("click", event => {
    if (!event.target.closest(".search")) {
        suggestionsList.innerHTML = "";
    }
});

// --- Creation Date Range slider ---
const minYearRange = document.getElementById("minYearRange");
const maxYearRange = document.getElementById("maxYearRange");
const minYearInput = document.getElementById("minYearInput");
const maxYearInput = document.getElementById("maxYearInput");
const yearprogress = document.getElementById("yearprogress"); // coloured fill bar

// --- Album Year Range slider ---
const minRange = document.getElementById("minRange");
const maxRange = document.getElementById("maxRange");
const minInput = document.getElementById("minInput");
const maxInput = document.getElementById("maxInput");
const progress = document.getElementById("progress"); // coloured fill bar

// Cache the min/max bounds (same for both sliders: 1960–2020)
const yearmin = parseInt(minYearRange.min);
const yearmax = parseInt(minYearRange.max);
const min = parseInt(minRange.min);
const max = parseInt(minRange.max);

// updateYearProgress recalculates and applies the fill bar position for the Creation Date slider.
function updateYearProgress() {
    const left = ((minYearRange.value - yearmin) / (yearmax - yearmin)) * 100;
    const right = ((maxYearRange.value - yearmin) / (yearmax - yearmin)) * 100;
    yearprogress.style.left = left + "%";
    yearprogress.style.width = (right - left) + "%";
}

// Sync the Creation Date range thumb → number input and update the fill bar
minYearRange.addEventListener("input", () => {
    if (parseInt(minYearRange.value) > parseInt(maxYearRange.value)) minYearRange.value = maxYearRange.value;
    minYearInput.value = minYearRange.value;
    updateYearProgress();
});

maxYearRange.addEventListener("input", () => {
    if (parseInt(maxYearRange.value) < parseInt(minYearRange.value)) maxYearRange.value = minYearRange.value;
    maxYearInput.value = maxYearRange.value;
    updateYearProgress();
});

// Sync the Creation Date number input → range thumb and update the fill bar
minYearInput.addEventListener("input", () => {
    if (parseInt(minYearInput.value) > parseInt(maxYearInput.value)) minYearInput.value = maxYearInput.value;
    minYearRange.value = minYearInput.value;
    updateYearProgress();
});

maxYearInput.addEventListener("input", () => {
    if (parseInt(maxYearInput.value) < parseInt(minYearInput.value)) maxYearInput.value = minYearInput.value;
    maxYearRange.value = maxYearInput.value;
    updateYearProgress();
});

// Initialise the Creation Date fill bar on page load
updateYearProgress();

// updateProgress recalculates and applies the fill bar position for the Album Year slider.
function updateProgress() {
    const left = ((minRange.value - min) / (max - min)) * 100;
    const right = ((maxRange.value - min) / (max - min)) * 100;
    progress.style.left = left + "%";
    progress.style.width = (right - left) + "%";
}

// Sync the Album Year range thumb → number input and update the fill bar
minRange.addEventListener("input", () => {
    if (parseInt(minRange.value) > parseInt(maxRange.value)) minRange.value = maxRange.value;
    minInput.value = minRange.value;
    updateProgress();
});

maxRange.addEventListener("input", () => {
    if (parseInt(maxRange.value) < parseInt(minRange.value)) maxRange.value = minRange.value;
    maxInput.value = maxRange.value;
    updateProgress();
});

// Sync the Album Year number input → range thumb and update the fill bar
minInput.addEventListener("input", () => {
    if (parseInt(minInput.value) > parseInt(maxInput.value)) minInput.value = maxInput.value;
    minRange.value = minInput.value;
    updateProgress();
});

maxInput.addEventListener("input", () => {
    if (parseInt(maxInput.value) < parseInt(minInput.value)) maxInput.value = minInput.value;
    maxRange.value = maxInput.value;
    updateProgress();
});

// Initialise the Album Year fill bar on page load
updateProgress();

// --- Custom Location Dropdown ---
(function initCustomLocationSelect() {
    const root = document.getElementById("locationCustom");
    const nativeSelect = document.getElementById("location-select");

    if (!root || !nativeSelect) return;

    const trigger = root.querySelector(".custom-select-trigger");
    const valueEl = root.querySelector(".custom-select-value");
    const options = Array.from(root.querySelectorAll(".custom-select-option"));

    if (!trigger || !valueEl || options.length === 0) return;

    const closeMenu = () => {
        root.classList.remove("is-open");
        trigger.setAttribute("aria-expanded", "false");
    };

    const openMenu = () => {
        root.classList.add("is-open");
        trigger.setAttribute("aria-expanded", "true");
    };

    const selectValue = value => {
        nativeSelect.value = value;

        options.forEach(opt => {
            const isSelected = (opt.getAttribute("data-value") || "") === value;
            opt.classList.toggle("is-selected", isSelected);

            if (isSelected) {
                valueEl.textContent = opt.textContent.trim();
            }
        });
    };

    const initialValue = nativeSelect.value || "";
    selectValue(initialValue);

    trigger.addEventListener("click", () => {
        if (root.classList.contains("is-open")) {
            closeMenu();
        } else {
            openMenu();
        }
    });

    options.forEach(opt => {
        opt.addEventListener("click", () => {
            const value = opt.getAttribute("data-value") || "";
            selectValue(value);
            closeMenu();
        });
    });

    document.addEventListener("click", event => {
        if (!root.contains(event.target)) {
            closeMenu();
        }
    });
})();
