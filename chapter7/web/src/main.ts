import {create, fromBinary, toBinary} from "@bufbuild/protobuf";
import {PlayholdClient, type ClientStatus, type ModuleView} from "playhold-sdk";

import {RematchSchema, ScoreMessage, ScoreViewSchema, type ScoreView} from "./gen/tictactoe/v1/score_pb";
import {BoardViewSchema, Mark, Message, PlaySchema, Status, type BoardView} from "./gen/tictactoe/v1/tictactoe_pb";

// The rules stay on the server: this page shows the board the server sends, and sends a move when a cell is clicked.

function element<T extends HTMLElement>(id: string): T {
    const found = document.getElementById(id);
    if (!found) {
        throw new Error(`missing element: ${id}`);
    }
    return found as T;
}

const ui = {
    connection: element("connection"),
    message: element("message"),
    newGame: element<HTMLButtonElement>("new-game"),
    game: element("game"),
    link: element<HTMLAnchorElement>("link"),
    status: element("status"),
    clock: element("clock"),
    board: element("board"),
    score: element("score"),
    rematch: element<HTMLButtonElement>("rematch"),
};

// #region text

const connectionText: Record<ClientStatus, string> = {
    disconnected: "Connection lost. Reconnecting…",
    connecting: "Connecting…",
    connected: "",
    queued: "",
    joined: "",
    replaced: "You opened this game somewhere else, so it stopped here.",
    closed: "Disconnected.",
    updateRequired: "There's a newer version of this game. Reload to keep playing.",
};

const statusText: Record<Status, string> = {
    [Status.UNSPECIFIED]: "",
    [Status.WAITING]: "Waiting for the other player…",
    [Status.YOUR_TURN]: "Your turn.",
    [Status.THEIR_TURN]: "Their turn.",
    [Status.WON]: "You won!",
    [Status.LOST]: "You lost.",
    [Status.DRAW]: "A draw.",
};

const markText: Record<Mark, string> = {[Mark.UNSPECIFIED]: "", [Mark.X]: "X", [Mark.O]: "O"};

/** How the game stands, saying so when it ended on time. */
function statusOf(board: BoardView): string {
    if (board.outOfTime) {
        return board.status === Status.WON ? "They ran out of time. You won!" : "You ran out of time.";
    }
    return statusText[board.status];
}

// #endregion text

// #region client

/**
 * The game's server: set when the page is built, such as VITE_SERVER_URL=wss://game.example/ws for a public one, or by
 * default port 8080 of the machine the page came from, where `go run ./server` listens.
 */
function serverUrl(): string {
    const configured: unknown = import.meta.env.VITE_SERVER_URL;
    if (typeof configured === "string" && configured !== "") {
        return configured;
    }
    return `ws://${location.hostname}:8080/ws`;
}

const client = new PlayholdClient({
    url: serverUrl(),
    onStatus: (status) => {
        ui.connection.textContent = connectionText[status];
    },
    onJoined: (code) => showRoom(code),
    onState: (views) => {
        render(boardOf(views));
        renderScore(scoreOf(views));
    },
    onError: (error) => {
        ui.message.textContent = error.message; // such as "it isn't your turn"
    },
});

// #endregion client

// #region views

/** The board's view, out of everything the player can see in the room: one view per module. */
function boardOf(views: ModuleView[]): BoardView | undefined {
    const view = views.find((v) => v.module === "board");
    return view && fromBinary(BoardViewSchema, view.view);
}

function render(board: BoardView | undefined): void {
    ui.status.textContent = board ? statusOf(board) : "";
    showClock(board?.turnMillisLeft ?? 0n);
    const over = board?.status === Status.WON || board?.status === Status.LOST || board?.status === Status.DRAW;
    ui.rematch.hidden = !over;
    ui.board.replaceChildren(
        ...(board?.cells ?? []).map((mark, cell) => {
            const button = document.createElement("button");
            button.textContent = markText[mark];
            button.disabled = board?.status !== Status.YOUR_TURN || mark !== Mark.UNSPECIFIED;
            button.addEventListener("click", () => play(cell));
            return button;
        }),
    );
}

// #endregion views

// #region score

/** The score's view: another module, in the same state as the board's. */
function scoreOf(views: ModuleView[]): ScoreView | undefined {
    const view = views.find((v) => v.module === "score");
    return view && fromBinary(ScoreViewSchema, view.view);
}

function renderScore(score: ScoreView | undefined): void {
    ui.score.textContent = score ? `You ${score.wins}, them ${score.losses}, draws ${score.draws}` : "";
}

/** Asks the score module for another game; it asks the board, through the board's keeper. */
function rematch(): void {
    ui.message.textContent = "";
    client.send("score", ScoreMessage.REMATCH, toBinary(RematchSchema, create(RematchSchema)));
}

ui.rematch.addEventListener("click", rematch);

// #endregion score

// #region clock

let ticking: number | undefined;

/**
 * Counts the turn's time down. The server only sends the time left with each view, so the page counts on from there
 * until the next one. The server's timer decides: this is only what the player sees.
 */
function showClock(millisLeft: bigint): void {
    clearInterval(ticking);
    ui.clock.textContent = "";
    if (millisLeft <= 0n) {
        return; // no game on
    }
    const deadline = performance.now() + Number(millisLeft);
    const tick = () => {
        const seconds = Math.max(0, Math.ceil((deadline - performance.now()) / 1000));
        ui.clock.textContent = `${seconds} s left`;
    };
    tick();
    ticking = setInterval(tick, 250);
}

// #endregion clock

// #region moves

function play(cell: number): void {
    ui.message.textContent = "";
    client.send("board", Message.PLAY, toBinary(PlaySchema, create(PlaySchema, {cell})));
}

/** Shows the room's code as a link to this page, for the other player to open. */
function showRoom(code: string): void {
    const link = new URL(location.href);
    link.search = new URLSearchParams({room: code}).toString();
    ui.link.href = link.href; // its text stays "this game's link": copy it with the browser's "Copy link"
    history.replaceState(null, "", link);
    ui.newGame.hidden = true;
    ui.game.hidden = false;
}

ui.newGame.addEventListener("click", () => client.createRoom()); // the code arrives in onJoined

client.connect();
const shared = new URLSearchParams(location.search).get("room");
if (shared) {
    client.join(shared); // opened from the other player's link
}

// #endregion moves