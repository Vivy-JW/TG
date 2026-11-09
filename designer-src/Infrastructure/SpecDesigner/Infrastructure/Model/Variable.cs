using System;
using System.Collections.Generic;
using System.Text.RegularExpressions;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200003A RID: 58
	public class Variable : IIntellisenseModel
	{
		// Token: 0x17000076 RID: 118
		// (get) Token: 0x0600015D RID: 349 RVA: 0x00006D23 File Offset: 0x00004F23
		// (set) Token: 0x0600015E RID: 350 RVA: 0x00006D2B File Offset: 0x00004F2B
		public bool IsFavorited { get; set; }

		// Token: 0x17000077 RID: 119
		// (get) Token: 0x0600015F RID: 351 RVA: 0x00006D34 File Offset: 0x00004F34
		public IntellisenseEnum Type
		{
			get
			{
				return IntellisenseEnum.Variable;
			}
		}

		// Token: 0x17000078 RID: 120
		// (get) Token: 0x06000160 RID: 352 RVA: 0x00006D37 File Offset: 0x00004F37
		public string FullName
		{
			get
			{
				return this.Name;
			}
		}

		// Token: 0x17000079 RID: 121
		// (get) Token: 0x06000161 RID: 353 RVA: 0x00006D3F File Offset: 0x00004F3F
		// (set) Token: 0x06000162 RID: 354 RVA: 0x00006D47 File Offset: 0x00004F47
		public string FunctionName { get; set; }

		// Token: 0x1700007A RID: 122
		// (get) Token: 0x06000163 RID: 355 RVA: 0x00006D50 File Offset: 0x00004F50
		// (set) Token: 0x06000164 RID: 356 RVA: 0x00006D58 File Offset: 0x00004F58
		public string Name { get; set; }

		// Token: 0x1700007B RID: 123
		// (get) Token: 0x06000165 RID: 357 RVA: 0x00006D61 File Offset: 0x00004F61
		// (set) Token: 0x06000166 RID: 358 RVA: 0x00006D69 File Offset: 0x00004F69
		public string Content
		{
			get
			{
				return this._content;
			}
			set
			{
				this._content = value.Trim();
			}
		}

		// Token: 0x1700007C RID: 124
		// (get) Token: 0x06000167 RID: 359 RVA: 0x00006D77 File Offset: 0x00004F77
		// (set) Token: 0x06000168 RID: 360 RVA: 0x00006D7F File Offset: 0x00004F7F
		public string Description
		{
			get
			{
				return this._description;
			}
			set
			{
				this._description = value.Trim();
			}
		}

		// Token: 0x06000169 RID: 361 RVA: 0x00006D8D File Offset: 0x00004F8D
		public Variable(string name, string content)
		{
			this.Name = name;
			this.Content = content;
			this.IsFavorited = false;
		}

		// Token: 0x0600016A RID: 362 RVA: 0x00007030 File Offset: 0x00005230
		public static IEnumerable<Variable> Parse(string variableString)
		{
			string strRegex = "(?<=(define)+\\s*).(?<name>\\w+).(\\s*)(?<content>.*$)";
			Regex regex = new Regex(strRegex, RegexOptions.IgnoreCase | RegexOptions.Multiline);
			if (regex.IsMatch(variableString))
			{
				MatchCollection mcName = regex.Matches(variableString);
				for (int matchIndex = 0; matchIndex < mcName.Count; matchIndex++)
				{
					string name = ((mcName[matchIndex].Groups.Count > 0) ? mcName[matchIndex].Groups["name"].Value.Trim() : string.Empty);
					if (!string.IsNullOrEmpty(name))
					{
						string content = ((mcName[matchIndex].Groups.Count > 0) ? mcName[matchIndex].Groups["content"].Value : string.Empty);
						string description = string.Empty;
						int markIndex = content.IndexOf('#');
						if (markIndex > 0)
						{
							description = content.Substring(markIndex + 1);
							content = content.Remove(markIndex);
						}
						yield return new Variable(name, content)
						{
							Description = description
						};
					}
				}
			}
			yield break;
		}

		// Token: 0x0600016B RID: 363 RVA: 0x0000704D File Offset: 0x0000524D
		public void Clear()
		{
		}

		// Token: 0x0600016C RID: 364 RVA: 0x0000704F File Offset: 0x0000524F
		public override string ToString()
		{
			return this.Description;
		}

		// Token: 0x04000098 RID: 152
		private string _content;

		// Token: 0x04000099 RID: 153
		private string _description;
	}
}
