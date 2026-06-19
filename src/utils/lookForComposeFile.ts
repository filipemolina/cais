import {
  COMPOSE_FILE_NAMES,
  type ComposeFileName,
} from "../consts/COMPOSE_FILE_NAMES";

type ConfigSetup = {
  fileName: ComposeFileName | null;
  file: Bun.BunFile | null;
};

async function lookForComposeFile(): Promise<ConfigSetup> {
  const composeFiles = COMPOSE_FILE_NAMES.map((fileName) => Bun.file(fileName));

  const filesExist = await Promise.all(composeFiles.map((f) => f.exists()));

  const fileIndex = filesExist.findIndex(Boolean);
  const fileName = fileIndex > -1 ? COMPOSE_FILE_NAMES[fileIndex] : undefined;
  const file = composeFiles[fileIndex];

  return {
    fileName: fileName || null,
    file: file || null,
  };
}

export default lookForComposeFile;
