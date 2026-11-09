using System;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x02000004 RID: 4
	public class FunctionAST : AST
	{
		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000011 RID: 17 RVA: 0x00002132 File Offset: 0x00000332
		// (set) Token: 0x06000012 RID: 18 RVA: 0x0000213A File Offset: 0x0000033A
		public FglToken Scope { get; set; }
	}
}
