using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Text;
using SpecDesignerPreference;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200002D RID: 45
	public class IntellisenseModel : IIntellisenseModel, IMethod
	{
		// Token: 0x06000119 RID: 281 RVA: 0x00005C84 File Offset: 0x00003E84
		public IntellisenseModel(string name)
		{
			this.Name = name;
		}

		// Token: 0x1700005B RID: 91
		// (get) Token: 0x0600011A RID: 282 RVA: 0x00005C9E File Offset: 0x00003E9E
		public string FullName
		{
			get
			{
				return string.Format("{0}({1})", this.Name, (this.GetParameterName().Count<string>() > 0) ? string.Join(",", this.GetParameterName()) : string.Empty);
			}
		}

		// Token: 0x1700005C RID: 92
		// (get) Token: 0x0600011B RID: 283 RVA: 0x00005CD5 File Offset: 0x00003ED5
		// (set) Token: 0x0600011C RID: 284 RVA: 0x00005CDD File Offset: 0x00003EDD
		[DefaultValue("")]
		public string Name { get; set; }

		// Token: 0x1700005D RID: 93
		// (get) Token: 0x0600011D RID: 285 RVA: 0x00005CE6 File Offset: 0x00003EE6
		// (set) Token: 0x0600011E RID: 286 RVA: 0x00005CEE File Offset: 0x00003EEE
		[DefaultValue("")]
		public string Description { get; set; }

		// Token: 0x1700005E RID: 94
		// (get) Token: 0x0600011F RID: 287 RVA: 0x00005CF8 File Offset: 0x00003EF8
		public string FunctionDescription
		{
			get
			{
				return string.Join(Environment.NewLine, new string[] { this.FullName, this.Description });
			}
		}

		// Token: 0x1700005F RID: 95
		// (get) Token: 0x06000120 RID: 288 RVA: 0x00005D29 File Offset: 0x00003F29
		// (set) Token: 0x06000121 RID: 289 RVA: 0x00005D31 File Offset: 0x00003F31
		public bool IsFavorited
		{
			get
			{
				return this._isFavorited;
			}
			set
			{
				this._isFavorited = value;
				if (this._isFavorited)
				{
					PreferenceManager.Current.Settings.AddFavoriteKeyword(this.Name);
					return;
				}
				PreferenceManager.Current.Settings.RemoveFavoriteKeyword(this.Name);
			}
		}

		// Token: 0x17000060 RID: 96
		// (get) Token: 0x06000122 RID: 290 RVA: 0x00005D6D File Offset: 0x00003F6D
		public IntellisenseEnum Type
		{
			get
			{
				return IntellisenseEnum.Function;
			}
		}

		// Token: 0x06000123 RID: 291 RVA: 0x00005D70 File Offset: 0x00003F70
		public void AppendParameter(IParameterModel parameter)
		{
			if (this.models == null)
			{
				this.models = new List<IParameterModel>();
			}
			this.models.Add(parameter);
		}

		// Token: 0x06000124 RID: 292 RVA: 0x00005D91 File Offset: 0x00003F91
		public void Clear()
		{
			this.Name = string.Empty;
			this.Description = string.Empty;
			this.models.Clear();
		}

		// Token: 0x06000125 RID: 293 RVA: 0x00006008 File Offset: 0x00004208
		public IEnumerable<IParameterModel> GetParameter(string name)
		{
			if (this.models == null)
			{
				this.models = new List<IParameterModel>();
			}
			IEnumerable<IParameterModel> matchedModel = null;
			if (string.IsNullOrEmpty(name))
			{
				matchedModel = this.models.AsEnumerable<IParameterModel>();
			}
			else
			{
				matchedModel = this.models.Where<IParameterModel>((IParameterModel m) => m.Name.Equals(name, StringComparison.CurrentCultureIgnoreCase));
			}
			foreach (IParameterModel i in matchedModel)
			{
				yield return i;
			}
			yield break;
		}

		// Token: 0x06000126 RID: 294 RVA: 0x00006240 File Offset: 0x00004440
		public IEnumerable<IParameterModel> GetParameter(UsageType usage)
		{
			if (this.models == null)
			{
				this.models = new List<IParameterModel>();
			}
			IEnumerable<IParameterModel> matchedModel = this.models.Where<IParameterModel>((IParameterModel m) => m.Usage == usage);
			foreach (IParameterModel i in matchedModel)
			{
				yield return i;
			}
			yield break;
		}

		// Token: 0x06000127 RID: 295 RVA: 0x0000626C File Offset: 0x0000446C
		private IEnumerable<string> GetParameterName()
		{
			return from p in this.GetParameter(UsageType.PARAMETER)
				select p.Name;
		}

		// Token: 0x17000061 RID: 97
		// (get) Token: 0x06000128 RID: 296 RVA: 0x000062C8 File Offset: 0x000044C8
		public string Content
		{
			get
			{
				if (this._content.Length == 0)
				{
					StringBuilder stringBuilder = new StringBuilder();
					stringBuilder.Append(this.Description);
					IEnumerable<string> enumerable = (IEnumerable<string>)(from p in this.GetParameter(UsageType.PARAMETER)
						select string.Format("{0}：{1}", p.Name, p.Description)).ToArray<string>();
					if (enumerable.Count<string>() > 0)
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.Append(string.Join(Environment.NewLine, enumerable));
					}
					IEnumerable<string> enumerable2 = (IEnumerable<string>)(from r in this.GetParameter(UsageType.RETURN)
						select string.Format("{0}：{1}", r.Name, r.Description)).ToArray<string>();
					if (enumerable2.Count<string>() > 0)
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.AppendLine("-------");
						stringBuilder.Append(string.Join(Environment.NewLine, enumerable2));
					}
					this._content = stringBuilder.ToString();
					stringBuilder.Clear();
				}
				return this._content;
			}
		}

		// Token: 0x06000129 RID: 297 RVA: 0x00006420 File Offset: 0x00004620
		public override string ToString()
		{
			StringBuilder stringBuilder = new StringBuilder();
			IEnumerable<string> enumerable = (IEnumerable<string>)(from p in this.GetParameter(UsageType.PARAMETER)
				select p.Name).ToArray<string>();
			stringBuilder.AppendLine(this.Name + "(" + string.Join(", ", enumerable) + ")");
			stringBuilder.Append(this.Description);
			IEnumerable<string> enumerable2 = (IEnumerable<string>)(from p in this.GetParameter(UsageType.PARAMETER)
				select string.Format("{0}：{1}", p.Name, p.Description)).ToArray<string>();
			if (enumerable2.Count<string>() > 0)
			{
				if (stringBuilder.Length > 0)
				{
					stringBuilder.Append(Environment.NewLine);
				}
				stringBuilder.Append(string.Join(Environment.NewLine, enumerable2));
			}
			IEnumerable<string> enumerable3 = (IEnumerable<string>)(from r in this.GetParameter(UsageType.RETURN)
				select string.Format("{0}：{1}", r.Name, r.Description)).ToArray<string>();
			if (enumerable3.Count<string>() > 0)
			{
				if (stringBuilder.Length > 0)
				{
					stringBuilder.Append(Environment.NewLine);
				}
				stringBuilder.AppendLine("Return:");
				stringBuilder.Append(string.Join(Environment.NewLine, enumerable3));
			}
			return stringBuilder.ToString();
		}

		// Token: 0x17000062 RID: 98
		// (get) Token: 0x0600012A RID: 298 RVA: 0x00006728 File Offset: 0x00004928
		public IEnumerable<IParameterModel> Parameters
		{
			get
			{
				if (this.models != null)
				{
					foreach (IParameterModel p in this.models)
					{
						if (p.Usage == UsageType.PARAMETER)
						{
							yield return p;
						}
					}
				}
				yield break;
			}
		}

		// Token: 0x0600012B RID: 299 RVA: 0x00006968 File Offset: 0x00004B68
		public IEnumerable<string> GetParameterDescription()
		{
			if (this.models != null)
			{
				foreach (IParameterModel p in this.models)
				{
					if (p.Usage == UsageType.PARAMETER)
					{
						if (p.Description.Length > 0)
						{
							yield return string.Format("{0}：{1}", p.Name, p.Description);
						}
						else
						{
							yield return p.Name;
						}
					}
				}
			}
			yield break;
		}

		// Token: 0x0600012C RID: 300 RVA: 0x00006988 File Offset: 0x00004B88
		public static IntellisenseModel Create(string name)
		{
			if (string.IsNullOrWhiteSpace(name))
			{
				return null;
			}
			return new IntellisenseModel(name)
			{
				IsFavorited = PreferenceManager.Current.Settings.IsFavoriteKeywordExist(name)
			};
		}

		// Token: 0x0400006E RID: 110
		private bool _isFavorited;

		// Token: 0x0400006F RID: 111
		private List<IParameterModel> models;

		// Token: 0x04000070 RID: 112
		private string _content = string.Empty;
	}
}
