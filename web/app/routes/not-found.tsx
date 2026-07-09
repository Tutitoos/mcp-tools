import { Link } from "react-router";
import { motion } from "motion/react";

export default function NotFoundRoute() {
  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        className="text-center"
      >
        <p className="font-mono text-sm text-muted-foreground">404</p>
        <h1 className="mt-2 text-3xl font-semibold">Página no encontrada</h1>
        <Link
          to="/"
          className="mt-4 inline-block text-sm text-foreground underline-offset-4 hover:underline"
        >
          Volver al dashboard
        </Link>
      </motion.div>
    </div>
  );
}