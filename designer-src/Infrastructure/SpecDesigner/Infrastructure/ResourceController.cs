using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Windows;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Helper;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerPreference;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x02000029 RID: 41
	public class ResourceController
	{
		// Token: 0x060000DA RID: 218 RVA: 0x00004DB2 File Offset: 0x00002FB2
		public static ResourceController GetInstance()
		{
			if (ResourceController._service == null)
			{
				ResourceController._service = new ResourceController();
			}
			return ResourceController._service;
		}

		// Token: 0x060000DB RID: 219 RVA: 0x00004DCC File Offset: 0x00002FCC
		private ResourceController()
		{
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeEvent>().Subscribe(new Action<PackageKey>(this.LoadFile));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.CloseFile));
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Subscribe(new Action<PackageKey>(this.SaveFile));
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Subscribe(new Action<string>(this.LoadBasicData));
			this.LoadBasicData(string.Empty);
		}

		// Token: 0x060000DC RID: 220 RVA: 0x00004EA8 File Offset: 0x000030A8
		public void Subscribe()
		{
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeEvent>().Unsubscribe(new Action<PackageKey>(this.LoadFile));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.CloseFile));
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Unsubscribe(delegate(string param)
			{
				this.LoadFunctionList();
			});
			EventController.GetInstance().GetEvent<RemoveFunctionModelEvent>().Unsubscribe(new Action<RemoveFunctionInformation>(this.OnRemoveFunctionModel));
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeEvent>().Subscribe(new Action<PackageKey>(this.LoadFile));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.CloseFile));
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Subscribe(delegate(string param)
			{
				this.LoadFunctionList();
			});
			EventController.GetInstance().GetEvent<RemoveFunctionModelEvent>().Subscribe(new Action<RemoveFunctionInformation>(this.OnRemoveFunctionModel));
		}

		// Token: 0x060000DD RID: 221 RVA: 0x00004F91 File Offset: 0x00003191
		public ProgramInformation GetProgramInfo(PackageKey key)
		{
			if (null == key)
			{
				return null;
			}
			if (this.ProgramMap.ContainsKey(key))
			{
				return this.ProgramMap[key];
			}
			return null;
		}

		// Token: 0x060000DE RID: 222 RVA: 0x00004FBC File Offset: 0x000031BC
		private void LoadFile(PackageKey key)
		{
			this.currentProgram = new ProgramInformation(key);
			this.ProgramMap.Add(key, this.currentProgram);
			this.GetProgramInfo(this.currentProgram.ProgramKey).AddPoint = SettingManager.Get().GetTzpManger(key).TAP;
			if (SettingManager.Get().GetTzpManger(key).DIFF_TAP != null)
			{
				this.GetProgramInfo(this.currentProgram.ProgramKey).DiffAddPoint = SettingManager.Get().GetTzpManger(key).DIFF_TAP.ToString();
			}
			try
			{
				this.GetProgramInfo(this.currentProgram.ProgramKey).TGL = SettingManager.Get().GetTzpManger(key).TGL;
			}
			catch (Exception ex)
			{
				DesignerMessageBox.Show(ex.Message, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK);
			}
			EventController.GetInstance().GetEvent<LoadedSettingEvent>().Publish(new LoadInformation(key, SettingManager.Get().GetTzpManger(key).IsDiff));
			this.currentProgram = null;
		}

		// Token: 0x060000DF RID: 223 RVA: 0x000050D4 File Offset: 0x000032D4
		private void CloseFile(PackageKey key)
		{
			if (this.ProgramMap.ContainsKey(key))
			{
				this.ProgramMap[key].Clear();
				this.ProgramMap.Remove(key);
			}
		}

		// Token: 0x060000E0 RID: 224 RVA: 0x00005104 File Offset: 0x00003304
		private void SaveFile(PackageKey key)
		{
			if (this.GetProgramInfo(key) != null)
			{
				SettingManager.Get().GetTzpManger(key).SaveCodeFile(this.GetProgramInfo(key).TGL);
				SettingManager.Get().GetTzpManger(key).SaveAddPoint(this.GetProgramInfo(key).AddPoint);
				SettingManager.Get().GetTzpManger(key).SaveUpdatedAddPoint(this.GetProgramInfo(key).UpdatedAddPoint);
				SettingManager.Get().GetTzpManger(key).SaveFullCode(this.GetProgramInfo(key).FullCode);
				EventController.GetInstance().GetEvent<ADPStatusUpdateEvent>().Publish(key);
			}
		}

		// Token: 0x060000E1 RID: 225 RVA: 0x0000519D File Offset: 0x0000339D
		private void OnRemoveFunctionModel(RemoveFunctionInformation info)
		{
			this.GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).Remove(info.ID);
		}

		// Token: 0x060000E2 RID: 226 RVA: 0x000051C4 File Offset: 0x000033C4
		private void Modify(ModifyInfomation info)
		{
			this.GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).Modify(info);
		}

		// Token: 0x060000E3 RID: 227 RVA: 0x000051E6 File Offset: 0x000033E6
		private void ModifyFunction(ModifyFunctionInformation info)
		{
			this.GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).FunctionModify(info);
		}

		// Token: 0x17000045 RID: 69
		// (get) Token: 0x060000E4 RID: 228 RVA: 0x00005208 File Offset: 0x00003408
		public ReadOnlyObservableCollection<IIntellisenseModel> GeneralFunctionSuggestion
		{
			get
			{
				if (this._generalFunctionSuggestion == null)
				{
					this._generalFunctionSuggestion = new ReadOnlyObservableCollection<IIntellisenseModel>(this._generalFunctions);
				}
				return this._generalFunctionSuggestion;
			}
		}

		// Token: 0x17000046 RID: 70
		// (get) Token: 0x060000E5 RID: 229 RVA: 0x00005229 File Offset: 0x00003429
		public ReadOnlyCollection<string> FglKeywords
		{
			get
			{
				return new ReadOnlyCollection<string>(this._fglKeywords);
			}
		}

		// Token: 0x060000E6 RID: 230 RVA: 0x00005238 File Offset: 0x00003438
		private void LoadBasicData(string empty)
		{
			try
			{
				this.LoadFunctionList();
				this.LoadCodeSample();
			}
			catch (Exception)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_BasicDataForCode") as string, Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.OK, MessageBoxImage.Exclamation);
			}
		}

		// Token: 0x060000E7 RID: 231 RVA: 0x000052B4 File Offset: 0x000034B4
		private void LoadFunctionList()
		{
			this._generalFunctions.Clear();
			if (this.KeywordSuggestion != null)
			{
				this.KeywordSuggestion.Clear();
			}
			if (this._fglKeywords != null)
			{
				this._fglKeywords.Clear();
			}
			if (SettingManager.Get().Info_Subroutines != null)
			{
				IntellisenseSourceFactory.Parse(SettingManager.Get().Info_Subroutines).ForEach(delegate(IIntellisenseModel i)
				{
					this._generalFunctions.Add(i);
				});
			}
			if (SettingManager.Get().Info_Libraries != null)
			{
				IntellisenseSourceFactory.Parse(SettingManager.Get().Info_Libraries).ForEach(delegate(IIntellisenseModel i)
				{
					this._generalFunctions.Add(i);
				});
			}
			int num = 0;
			foreach (string text in (PreferenceManager.Current.Settings.GetSelfKeyword() ?? new List<string>()))
			{
				SelfIntellisenseModel selfIntellisenseModel = SelfIntellisenseModel.Create(text);
				if (selfIntellisenseModel != null)
				{
					this._generalFunctions.Insert(num++, selfIntellisenseModel);
				}
			}
			this.ConvertKeywordToIntellisense();
		}

		// Token: 0x060000E8 RID: 232 RVA: 0x000053C0 File Offset: 0x000035C0
		private void ConvertKeywordToIntellisense()
		{
			if (string.IsNullOrEmpty(SettingManager.Get().Info_FGLKeyword))
			{
				return;
			}
			IEnumerable<XElement> enumerable = from query in XDocument.Parse(SettingManager.Get().Info_FGLKeyword).Descendants("KeywordSets")
				select (query);
			foreach (XElement xelement in enumerable)
			{
				string[] array = xelement.Value.Split(new char[] { ' ' });
				foreach (string text in array)
				{
					Keyword keyword = new Keyword(text);
					this.KeywordSuggestion.Add(keyword);
					this._fglKeywords.Add(text);
				}
			}
			if (SettingManager.Get().TopGlobals != null)
			{
				enumerable = from query in SettingManager.Get().TopGlobals.Descendants("var")
					select (query);
				foreach (XElement xelement2 in enumerable)
				{
					Keyword keyword2 = new Keyword(xelement2.Attribute("name").Value);
					keyword2.Description = Application.Current.FindResource("CE_ShowGlobals") as string;
					this.KeywordSuggestion.Add(keyword2);
				}
			}
		}

		// Token: 0x17000047 RID: 71
		// (get) Token: 0x060000E9 RID: 233 RVA: 0x00005578 File Offset: 0x00003778
		public ReadOnlyCollection<CodeSamepleModel> CodeSamples
		{
			get
			{
				return new ReadOnlyCollection<CodeSamepleModel>(this._samples);
			}
		}

		// Token: 0x060000EA RID: 234 RVA: 0x00005593 File Offset: 0x00003793
		private void LoadCodeSample()
		{
			this._samples.Clear();
			CodeSampleSourceFactory.Parse(SettingManager.Get().Info_CodeSample).ForEach(delegate(CodeSamepleModel i)
			{
				this._samples.Add(i);
			});
		}

		// Token: 0x060000EB RID: 235 RVA: 0x000055C0 File Offset: 0x000037C0
		public void DeleteSuggestion(string name)
		{
			for (int i = 0; i < this.GeneralFunctionSuggestion.Count; i++)
			{
				IIntellisenseModel intellisenseModel = this.GeneralFunctionSuggestion[i];
				if (intellisenseModel.Type == IntellisenseEnum.Self && intellisenseModel.Name == name)
				{
					this._generalFunctions.RemoveAt(i);
					PreferenceManager.Current.Settings.RemoveSelfKeyword(intellisenseModel.Name);
					return;
				}
			}
		}

		// Token: 0x060000EC RID: 236 RVA: 0x0000564C File Offset: 0x0000384C
		public void AddSuggestion(IIntellisenseModel model)
		{
			IEnumerable<IIntellisenseModel> enumerable = this.GeneralFunctionSuggestion.Where<IIntellisenseModel>((IIntellisenseModel m) => m.Name == model.Name);
			if (enumerable.Count<IIntellisenseModel>() > 0)
			{
				return;
			}
			this._generalFunctions.Insert(0, model);
			PreferenceManager.Current.Settings.AddSelfKeyword(model.Name);
		}

		// Token: 0x0400005F RID: 95
		private static ResourceController _service = new ResourceController();

		// Token: 0x04000060 RID: 96
		private ProgramInformation currentProgram;

		// Token: 0x04000061 RID: 97
		private Dictionary<PackageKey, ProgramInformation> ProgramMap = new Dictionary<PackageKey, ProgramInformation>(new PackageKey.PackageKeyComparer());

		// Token: 0x04000062 RID: 98
		public List<IIntellisenseModel> KeywordSuggestion = new List<IIntellisenseModel>();

		// Token: 0x04000063 RID: 99
		private ObservableCollection<IIntellisenseModel> _generalFunctions = new ObservableCollection<IIntellisenseModel>();

		// Token: 0x04000064 RID: 100
		private ReadOnlyObservableCollection<IIntellisenseModel> _generalFunctionSuggestion;

		// Token: 0x04000065 RID: 101
		private List<string> _fglKeywords = new List<string>();

		// Token: 0x04000066 RID: 102
		private List<CodeSamepleModel> _samples = new List<CodeSamepleModel>();
	}
}
