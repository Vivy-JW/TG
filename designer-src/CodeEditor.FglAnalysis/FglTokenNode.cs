using System;
using System.Collections.Generic;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x02000008 RID: 8
	public class FglTokenNode
	{
		// Token: 0x1700000C RID: 12
		// (get) Token: 0x06000020 RID: 32 RVA: 0x0000300F File Offset: 0x0000120F
		public List<FglTokenNode> Children
		{
			get
			{
				return this._child;
			}
		}

		// Token: 0x1700000D RID: 13
		// (get) Token: 0x06000021 RID: 33 RVA: 0x00003017 File Offset: 0x00001217
		// (set) Token: 0x06000022 RID: 34 RVA: 0x0000301F File Offset: 0x0000121F
		public FglTokenNode Parent
		{
			get
			{
				return this._parent;
			}
			set
			{
				this._parent = value;
			}
		}

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x06000023 RID: 35 RVA: 0x00003028 File Offset: 0x00001228
		// (set) Token: 0x06000024 RID: 36 RVA: 0x00003030 File Offset: 0x00001230
		public FglToken Token
		{
			get
			{
				return this._token;
			}
			set
			{
				this._token = value;
				this.Content = this._token.Text;
			}
		}

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x06000025 RID: 37 RVA: 0x0000304A File Offset: 0x0000124A
		// (set) Token: 0x06000026 RID: 38 RVA: 0x00003052 File Offset: 0x00001252
		public string Content { get; set; }

		// Token: 0x06000027 RID: 39 RVA: 0x0000305B File Offset: 0x0000125B
		public FglTokenNode(FglToken token)
		{
			this.Token = token.Clone();
		}

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x06000028 RID: 40 RVA: 0x0000307A File Offset: 0x0000127A
		public int Count
		{
			get
			{
				return this._child.Count;
			}
		}

		// Token: 0x06000029 RID: 41 RVA: 0x00003088 File Offset: 0x00001288
		public override string ToString()
		{
			string text = this.Token.Text;
			if (this.Children.Count > 0)
			{
				string text2 = string.Empty;
				for (int i = 0; i < this.Children.Count; i++)
				{
					FglTokenNode fglTokenNode = this.Children[i];
					if (i > 0)
					{
						text2 = ((fglTokenNode.Token.Type == TokenType.PEROID_TOKEN || this.Children[i - 1].Token.Type == TokenType.PEROID_TOKEN) ? string.Format("{0}{1}", text2, fglTokenNode.Token.Text) : string.Format("{0} {1}", text2, fglTokenNode.Token.Text));
					}
					else
					{
						text2 = ((fglTokenNode.Token.Type == TokenType.PEROID_TOKEN) ? string.Format("{0}{1}", text2, fglTokenNode.Token.Text) : string.Format("{0} {1}", text2, fglTokenNode.Token.Text));
					}
				}
				text += text2;
			}
			return text;
		}

		// Token: 0x0400001C RID: 28
		private List<FglTokenNode> _child = new List<FglTokenNode>();

		// Token: 0x0400001D RID: 29
		private FglTokenNode _parent;

		// Token: 0x0400001E RID: 30
		private FglToken _token;
	}
}
