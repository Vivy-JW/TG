using System;
using System.Collections.Generic;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x02000003 RID: 3
	public class AST
	{
		// Token: 0x06000005 RID: 5 RVA: 0x000020A6 File Offset: 0x000002A6
		public AST()
		{
			this.childNodes = new List<AST>();
			this.EndNode = null;
			this.ParentNode = null;
			this.NodeKind = -1;
			this.TokenNode = null;
		}

		// Token: 0x17000003 RID: 3
		// (get) Token: 0x06000006 RID: 6 RVA: 0x000020D5 File Offset: 0x000002D5
		public IList<AST> ChildNodes
		{
			get
			{
				return this.childNodes;
			}
		}

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x06000007 RID: 7 RVA: 0x000020DD File Offset: 0x000002DD
		// (set) Token: 0x06000008 RID: 8 RVA: 0x000020E5 File Offset: 0x000002E5
		public int NodeKind { get; set; }

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x06000009 RID: 9 RVA: 0x000020EE File Offset: 0x000002EE
		// (set) Token: 0x0600000A RID: 10 RVA: 0x000020F6 File Offset: 0x000002F6
		public FglTokenNode TokenNode { get; set; }

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x0600000B RID: 11 RVA: 0x000020FF File Offset: 0x000002FF
		// (set) Token: 0x0600000C RID: 12 RVA: 0x00002107 File Offset: 0x00000307
		public int Line { get; set; }

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x0600000D RID: 13 RVA: 0x00002110 File Offset: 0x00000310
		// (set) Token: 0x0600000E RID: 14 RVA: 0x00002118 File Offset: 0x00000318
		public AST EndNode { get; set; }

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x0600000F RID: 15 RVA: 0x00002121 File Offset: 0x00000321
		// (set) Token: 0x06000010 RID: 16 RVA: 0x00002129 File Offset: 0x00000329
		public AST ParentNode { get; set; }

		// Token: 0x04000003 RID: 3
		private List<AST> childNodes;
	}
}
