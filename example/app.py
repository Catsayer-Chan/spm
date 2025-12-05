import asyncio
import logging
import signal

import tornado
import tornado.ioloop
import tornado.web


def handle_signal(sig):
    loop = asyncio.get_running_loop()
    for task in asyncio.all_tasks(loop=loop):
        task.cancel()

    print(f'Got signal: {sig!s}, shutting down.')
    loop.remove_signal_handler(signal.SIGTERM)
    loop.add_signal_handler(signal.SIGINT, lambda: None)


class MainHandler(tornado.web.RequestHandler):
    def get(self):
        self.write("Hello, world")


def make_app():
    access_log = logging.getLogger("tornado.access")

    console_handler = logging.StreamHandler()
    formatter = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')
    console_handler.setFormatter(formatter)
    access_log.addHandler(console_handler)

    access_log.setLevel(logging.DEBUG)
    access_log.propagate = False

    return tornado.web.Application([
        (r"/", MainHandler),
    ])


async def main():
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM, signal.SIGQUIT):
        loop.add_signal_handler(sig, handle_signal, sig)

    app = make_app()
    try:
        print("Start tornado server on :3000")
        app.listen(3000)
        await asyncio.Event().wait()
    except asyncio.CancelledError:
        await asyncio.sleep(1)


if __name__ == "__main__":
    asyncio.run(main())

