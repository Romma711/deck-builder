import {Link} from "wouter"

export function Header () {
    return(
        <>
            <header className="w-screen h-[65px] bg-slate-600">
                <nav className="flex justify-start items-center w-[75%] h-full bg-slate-300 m-auto">
                    <div className="justify-start w-[200px]">
                        <Link>Logo</Link>
                    </div>
                    <ul className="flex w-[600px] bg-slate-500">
                            <li><Link>Ver Decks</Link></li>
                            <li><Link>Mis Decks</Link></li>
                            <li><Link>Crear Deck</Link></li>
                            <li><Link>Log in / Sign in</Link></li>
                    </ul>
                    <form action="" method="get">
                        <input type="text" placeholder="Buscar carta" />
                    </form>
                </nav>
            </header>
        </>
    )
}
