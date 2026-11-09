using System;
using CodeEditor.FglAnalysis;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000002 RID: 2
	public class FglParserQuickHelper
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public static ParseResult ParseFunction(string functionToParse)
		{
			ParseResult parseResult = new ParseResult();
			try
			{
				FglScanner fglScanner = new FglScanner(new FglReader(functionToParse));
				FglParser fglParser = new FglParser(fglScanner);
				AST ast = fglParser.Parse();
				if (ast.ChildNodes.Count > 0)
				{
					AST ast2 = ast.ChildNodes[0];
					parseResult.Name = ast2.TokenNode.ToString();
					parseResult.StartLineNumber = ast2.Line;
				}
			}
			catch
			{
				parseResult = null;
			}
			return parseResult;
		}
	}
}
