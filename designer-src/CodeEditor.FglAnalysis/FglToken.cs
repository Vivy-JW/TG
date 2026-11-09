using System;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x0200000B RID: 11
	public class FglToken
	{
		// Token: 0x17000012 RID: 18
		// (get) Token: 0x06000045 RID: 69 RVA: 0x0000494C File Offset: 0x00002B4C
		// (set) Token: 0x06000046 RID: 70 RVA: 0x00004954 File Offset: 0x00002B54
		public TokenType Type { get; set; }

		// Token: 0x17000013 RID: 19
		// (get) Token: 0x06000047 RID: 71 RVA: 0x0000495D File Offset: 0x00002B5D
		// (set) Token: 0x06000048 RID: 72 RVA: 0x00004965 File Offset: 0x00002B65
		public string Text { get; set; }

		// Token: 0x06000049 RID: 73 RVA: 0x0000496E File Offset: 0x00002B6E
		public FglToken(TokenType type, string text)
		{
			this.Type = type;
			this.Text = text;
		}

		// Token: 0x0600004A RID: 74 RVA: 0x00004984 File Offset: 0x00002B84
		public FglToken()
		{
		}

		// Token: 0x0600004B RID: 75 RVA: 0x0000498C File Offset: 0x00002B8C
		public FglToken Clone()
		{
			return base.MemberwiseClone() as FglToken;
		}
	}
}
