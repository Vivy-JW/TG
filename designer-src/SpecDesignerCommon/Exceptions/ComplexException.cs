using System;

namespace SpecDesignerCommon.Exceptions
{
	// Token: 0x02000073 RID: 115
	public class ComplexException : Exception
	{
		// Token: 0x0600046B RID: 1131 RVA: 0x00014199 File Offset: 0x00012399
		public ComplexException(string mainMessage, string detailMessage)
		{
			this._message = mainMessage;
			this._detailMessage = detailMessage;
		}

		// Token: 0x1700012D RID: 301
		// (get) Token: 0x0600046C RID: 1132 RVA: 0x000141AF File Offset: 0x000123AF
		public override string Message
		{
			get
			{
				return this._message;
			}
		}

		// Token: 0x1700012E RID: 302
		// (get) Token: 0x0600046D RID: 1133 RVA: 0x000141B7 File Offset: 0x000123B7
		public string DetailMessage
		{
			get
			{
				return this._detailMessage;
			}
		}

		// Token: 0x040001BA RID: 442
		private string _message;

		// Token: 0x040001BB RID: 443
		private string _detailMessage;
	}
}
